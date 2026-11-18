package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var migration embed.FS

type store struct{ db *pgxpool.Pool }

type worldState struct {
	Last int64  `json:"last_committed_tick,string"`
	Due  int64  `json:"due_tick,string"`
	Mode string `json:"mode"`
}

type commandState struct {
	ID     string `json:"command_id"`
	Target int64  `json:"target_tick,string"`
	Result *int64 `json:"result_tick,string,omitempty"`
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
}

func (s store) migrate(ctx context.Context) error {
	sql, err := migration.ReadFile("schema.sql")
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, string(sql))
	return err
}

func (s store) seed(ctx context.Context, world, account, token string) error {
	if len(token) < 24 || world == "" || account == "" {
		return errors.New("world, account and token of at least 24 characters required")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "INSERT INTO worlds(id) VALUES($1) ON CONFLICT DO NOTHING", world); err != nil {
		return err
	}
	var last int64
	if err = tx.QueryRow(ctx, "SELECT last_tick FROM worlds WHERE id=$1 FOR UPDATE", world).Scan(&last); err != nil {
		return err
	}
	initial := company{Cash: initialCash, LastTick: last, SupplyLot: accountSupply(account) != nil}
	if isMill(account) {
		initial.MillFlour = flourProduced
		if account == remoteMill {
			initial.MillLot = "remote-" + millOfferID
		}
	}
	state, err := json.Marshal(initial)
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(token))
	if _, err = tx.Exec(ctx, `INSERT INTO companies(world,account,token_hash,state) VALUES($1,$2,$3,$4)
     ON CONFLICT(world,account) DO UPDATE SET token_hash=excluded.token_hash`, world, account, hash[:], state); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s store) authenticate(ctx context.Context, world, token string) (string, error) {
	var account string
	hash := sha256.Sum256([]byte(token))
	err := s.db.QueryRow(ctx, "SELECT account FROM companies WHERE world=$1 AND token_hash=$2", world, hash[:]).Scan(&account)
	return account, err
}

func scopeError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errScope
	}
	return err
}

func (s store) accept(ctx context.Context, world, account string, cmd command) (commandState, error) {
	result := commandState{ID: cmd.ID, Status: "accepted"}
	body, err := json.Marshal(cmd)
	if err != nil {
		return result, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	var last int64
	var recovering bool
	err = tx.QueryRow(ctx, "SELECT last_tick,recovering FROM worlds WHERE id=$1 FOR UPDATE", world).Scan(&last, &recovering)
	if err != nil {
		return result, scopeError(err)
	}
	var same bool
	err = tx.QueryRow(ctx, `SELECT target,body=$4::jsonb FROM commands WHERE world=$1 AND account=$2 AND id=$3`,
		world, account, cmd.ID, body).Scan(&result.Target, &same)
	if err == nil {
		if !same {
			return result, errConflict
		}
		return result, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return result, err
	}
	if recovering {
		return result, errRecovering
	}
	result.Target = last + 1
	_, err = tx.Exec(ctx, `INSERT INTO commands(world,account,id,body,target) VALUES($1,$2,$3,$4,$5)`,
		world, account, cmd.ID, body, result.Target)
	if err != nil {
		return result, err
	}
	return result, tx.Commit(ctx)
}

func (s store) lookup(ctx context.Context, world, account, id string) (commandState, error) {
	result := commandState{ID: id}
	err := s.db.QueryRow(ctx, "SELECT target,status,reason FROM commands WHERE world=$1 AND account=$2 AND id=$3",
		world, account, id).Scan(&result.Target, &result.Status, &result.Reason)
	if result.Status != "accepted" && err == nil {
		result.Result = &result.Target
	}
	return result, scopeError(err)
}

func (s store) company(ctx context.Context, world, account string) (company, error) {
	var raw []byte
	err := s.db.QueryRow(ctx, "SELECT state FROM companies WHERE world=$1 AND account=$2", world, account).Scan(&raw)
	if err != nil {
		return company{}, scopeError(err)
	}
	return decodeCompany(raw)
}

func (s store) status(ctx context.Context, world string) (worldState, error) {
	var state worldState
	err := s.db.QueryRow(ctx, `SELECT last_tick,due_tick,CASE WHEN recovering THEN 'recovering' ELSE 'running' END
  FROM worlds WHERE id=$1`, world).Scan(&state.Last, &state.Due, &state.Mode)
	return state, scopeError(err)
}

func (s store) setDue(ctx context.Context, world string, due int64) error {
	tag, err := s.db.Exec(ctx, `UPDATE worlds SET due_tick=GREATEST(due_tick,$2),
  recovering=GREATEST(due_tick,$2)>last_tick WHERE id=$1`, world, due)
	if err == nil && tag.RowsAffected() != 1 {
		return errScope
	}
	return err
}

type pending struct {
	account string
	cmd     command
}

func loadCommands(ctx context.Context, tx pgx.Tx, world string, tick int64) ([]pending, error) {
	rows, err := tx.Query(ctx, `SELECT account,body FROM commands
  WHERE world=$1 AND target=$2 AND status='accepted' ORDER BY account,id`, world, tick)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var commands []pending
	for rows.Next() {
		var next pending
		var body []byte
		if err = rows.Scan(&next.account, &body); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(body, &next.cmd); err != nil {
			return nil, err
		}
		commands = append(commands, next)
	}
	return commands, rows.Err()
}

func loadCompanies(ctx context.Context, tx pgx.Tx, world string) (map[string]company, error) {
	rows, err := tx.Query(ctx, "SELECT account,state FROM companies WHERE world=$1", world)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[string]company{}
	for rows.Next() {
		var account string
		var raw []byte
		if err = rows.Scan(&account, &raw); err != nil {
			return nil, err
		}
		c, err := decodeCompany(raw)
		if err != nil {
			return nil, err
		}
		result[account] = c
	}
	return result, rows.Err()
}

func (s store) tick(ctx context.Context, world string, target int64, beforeCommit func() error) error {
	return s.tickWithProduction(ctx, world, target, beforeCommit, produce)
}

func (s store) tickWithProduction(ctx context.Context, world string, target int64,
	beforeCommit func() error, production func(*company, int64) []event) error {
	if target < 1 {
		return errTickOrder
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var last int64
	if err = tx.QueryRow(ctx, "SELECT last_tick FROM worlds WHERE id=$1 FOR UPDATE", world).Scan(&last); err != nil {
		return scopeError(err)
	}
	if target <= last {
		return nil
	}
	if target != last+1 {
		return errTickOrder
	}
	companies, err := loadCompanies(ctx, tx, world)
	if err != nil {
		return err
	}
	commands, err := loadCommands(ctx, tx, world, target)
	if err != nil {
		return err
	}
	events := make(map[string][]event)
	for _, p := range commands {
		c, ok := companies[p.account]
		if !ok {
			return fmt.Errorf("missing company for command %s", p.cmd.ID)
		}
		var deltas []event
		var reason string
		if accountSupply(p.account) != nil {
			reason = "SUPPLIER_ACCOUNT"
		} else if p.cmd.Type == "buy_chain_supply" && !isMill(p.account) {
			offer := supplyByID(p.cmd.Payload.Offer)
			sellerID := offer.account()
			seller := companies[sellerID]
			var sellerEvents []event
			deltas, sellerEvents, reason = buyChainSupply(&c, &seller, offer, p.cmd.Payload.Offer, p.cmd.Payload.Limit, target)
			if reason == "" {
				companies[sellerID] = seller
				events[sellerID] = append(events[sellerID], sellerEvents...)
			}
		} else if p.cmd.Type == "buy_mill_flour" {
			sellerID := millSeller(p.cmd.Payload.Offer)
			seller := companies[sellerID]
			var sellerEvents []event
			deltas, sellerEvents, reason = buyMillFlour(&c, &seller, p.account, p.cmd.Payload.Offer, p.cmd.Payload.Limit, target)
			if reason == "" {
				companies[sellerID] = seller
				events[sellerID] = append(events[sellerID], sellerEvents...)
			}
		} else if p.cmd.Type == "buy_mill_grain" || p.cmd.Type == "list_mill_flour" {
			if isMill(p.account) {
				deltas, reason = c.applyMillCommand(p.account, p.cmd.Type, target)
			} else {
				reason = "MILL_ACCOUNT_REQUIRED"
			}
		} else if isMill(p.account) {
			reason = "BAKERY_ACCOUNT_REQUIRED"
		} else {
			deltas, reason = applyCommand(&c, p.cmd, target)
		}
		status := "applied"
		if reason != "" {
			status = "rejected"
		}
		companies[p.account] = c
		events[p.account] = append(events[p.account], deltas...)
		if _, err = tx.Exec(ctx, `UPDATE commands SET status=$4,reason=$5
   WHERE world=$1 AND account=$2 AND id=$3`, world, p.account, p.cmd.ID, status, reason); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, "INSERT INTO ticks(world,number) VALUES($1,$2)", world, target); err != nil {
		return err
	}
	batch := &pgx.Batch{}
	for account, c := range companies {
		accountEvents := events[account]
		if offer := accountSupply(account); offer != nil {
			accountEvents = append(accountEvents, c.restockSupply(offer, target)...)
		} else if isMill(account) {
			accountEvents = append(accountEvents, c.produceMill(target)...)
		} else {
			accountEvents = append(accountEvents, c.deliverMillFlour(target)...)
			accountEvents = append(accountEvents, c.advanceMillOrder(target)...)
			accountEvents = append(accountEvents, c.deliverChain(target)...)
			accountEvents = append(accountEvents, c.advanceChainOrder(target)...)
			accountEvents = append(accountEvents, production(&c, target)...)
		}
		c.LastTick = target
		state, err := json.Marshal(c)
		if err != nil {
			return err
		}
		batch.Queue("UPDATE companies SET state=$3 WHERE world=$1 AND account=$2", world, account, state)
		for ordinal, e := range accountEvents {
			raw, err := json.Marshal(e.Movements)
			if err != nil {
				return err
			}
			batch.Queue(`INSERT INTO ledger(world,account,tick,ordinal,event,movements)
			    VALUES($1,$2,$3,$4,$5,$6)`, world, account, target, ordinal, e.Name, raw)
		}
	}
	if batch.Len() > 0 {
		if err := tx.SendBatch(ctx, batch).Close(); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE worlds SET last_tick=$2,due_tick=GREATEST(due_tick,$2),
   recovering=due_tick>$2 WHERE id=$1`, world, target); err != nil {
		return err
	}
	if beforeCommit != nil {
		if err = beforeCommit(); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
