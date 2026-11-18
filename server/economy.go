package main

import (
	"encoding/json"
	"errors"
	"strconv"
)

const (
	initialCash     = 200_000
	grainCost       = 30_000
	ingredientsCost = 18_000
	transportCost   = 12_000
	lineCost        = 14_000
	flourProduced   = 7_400
	flourUsed       = 7_200
	waterUsed       = 5_440
	saltUsed        = 144
	yeastUsed       = 112
	breadCount      = 24
	breadPrice      = 4_500
	packagingCost   = 2_400
)

type company struct {
	NextMill          *advanceOrder `json:"next_mill_order,omitempty"`
	SupplyRestockTick int64         `json:"supply_restock_tick,string"`
	SupplyLotTick     int64         `json:"supply_lot_tick,string"`
	SupplyCost        int64         `json:"supply_cost_kopecks,string"`
	SupplyLot         bool          `json:"supply_lot"`
	InTransitUntil    int64         `json:"mill_delivery_tick,string"`
	MillPurchaseCost  int64         `json:"mill_purchase_cost_kopecks,string"`
	MillGrain         int64         `json:"mill_grain_g,string"`
	MillGrainReady    int64         `json:"mill_grain_ready_tick,string"`
	MillLot           string        `json:"mill_lot_id,omitempty"`
	MillUnlisted      bool          `json:"mill_unlisted"`
	MillCost          int64         `json:"mill_cost_kopecks,string"`

	MillFlour         int64        `json:"mill_flour_g,string"`
	CompletedBatches  int64        `json:"completed_batches,string"`
	OpeningFlourValue int64        `json:"opening_flour_value_kopecks,string"`
	Batch             *chainBatch  `json:"chain_batch,omitempty"`
	LoadCells         *[10]company `json:"load_cells,omitempty"`
	Cash              int64        `json:"cash_kopecks,string"`
	Grain             int64        `json:"grain_g,string"`
	Water             int64        `json:"water_g,string"`
	Salt              int64        `json:"salt_g,string"`
	Yeast             int64        `json:"yeast_g,string"`
	PaperPack         int64        `json:"paper_pack_count,string"`
	PackedBread       int64        `json:"packed_bread_count,string"`
	PackOrder         bool         `json:"pack_order"`
	BreadLot          string       `json:"bread_lot_id,omitempty"`
	Flour             int64        `json:"flour_g,string"`
	FlourValue        int64        `json:"flour_value_kopecks,string"`
	Dough             int64        `json:"dough_g,string"`
	Bread             int64        `json:"bread_count,string"`
	GrainReady        int64        `json:"grain_ready_tick,string"`
	FlourReady        int64        `json:"flour_ready_tick,string"`
	BreadReady        int64        `json:"bread_ready_tick,string"`
	Line              bool         `json:"line_enabled"`
	Revenue           int64        `json:"revenue_kopecks,string"`
	Cost              int64        `json:"cost_kopecks,string"`
	Profit            int64        `json:"profit_kopecks,string"`
	LastTick          int64        `json:"last_committed_tick,string"`
}

type payload struct {
	Offer      string `json:"offer_id,omitempty"`
	Quantity   int64  `json:"quantity,string,omitempty"`
	Limit      int64  `json:"max_price_kopecks,string,omitempty"`
	Enterprise string `json:"enterprise_id,omitempty"`
	Cell       string `json:"cell_id,omitempty"`
	Profile    string `json:"profile_id,omitempty"`
	Lot        string `json:"lot_id,omitempty"`
	Price      int64  `json:"price_kopecks,string,omitempty"`
}

type command struct {
	ID      string  `json:"command_id"`
	Type    string  `json:"type"`
	Payload payload `json:"payload"`
}

func (cmd command) valid() bool {
	p := cmd.Payload
	if len(cmd.ID) == 0 || len(cmd.ID) > 128 {
		return false
	}
	switch cmd.Type {
	case "order_raw":
		return (p.Offer == "pilot-inputs-v1" || p.Offer == "pilot-inputs-v2" || chainByID(p.Offer) != nil) &&
			p == (payload{Offer: p.Offer, Quantity: 1, Limit: p.Limit}) && p.Limit > 0
	case "buy_chain_supply":
		return supplyByID(p.Offer) != nil && p == (payload{Offer: p.Offer, Quantity: 1, Limit: p.Limit}) && p.Limit > 0
	case "buy_mill_flour":
		return validMillLot(p.Offer) && p == (payload{Offer: p.Offer, Quantity: flourProduced, Limit: p.Limit}) && p.Limit > 0
	case "buy_mill_grain", "list_mill_flour":
		return p == (payload{})
	case "configure_line":
		return p == (payload{Enterprise: "pilot-bakery", Cell: "4", Profile: "test-bread-v1"})
	case "offer_output":
		if p != (payload{Lot: p.Lot, Quantity: p.Quantity, Price: p.Price}) || len(p.Lot) > 128 || p.Lot == "" {
			return false
		}
		for _, chain := range pilotChains {
			if p.Quantity == chain.Quantity && p.Price == chain.Price {
				return true
			}
		}
		return p.Quantity == breadCount && p.Price == breadPrice
	}
	return false
}

type event struct {
	Name      string
	Movements map[string]string
}

func movement(name string, deltas map[string]int64) event {
	serialized := make(map[string]string, len(deltas))
	for key, value := range deltas {
		serialized[key] = strconv.FormatInt(value, 10)
	}
	return event{name, serialized}
}

func (c *company) batchActive() bool {
	return (c.GrainReady != 0 || c.Batch != nil) && (c.BreadReady == 0 || c.BreadLot != "")
}

func applyCommand(c *company, cmd command, tick int64) ([]event, string) {
	switch cmd.Type {
	case "configure_line":
		c.Line = true
	case "order_raw":
		if c.batchActive() || c.NextMill != nil || (c.Batch != nil && c.Batch.Next != nil) {
			return nil, "PILOT_ORDER_EXISTS"
		}
		if chain := chainByID(cmd.Payload.Offer); chain != nil {
			if c.GrainReady != 0 || (c.Batch != nil && c.Batch.ID != chain.ID) {
				return nil, "PILOT_CHAIN_LOCKED"
			}
			return c.buyChain(chain, cmd.Payload.Limit, tick)
		}
		if c.Batch != nil {
			return nil, "PILOT_CHAIN_LOCKED"
		}
		purchaseCost := int64(grainCost + ingredientsCost)
		packs := int64(0)
		if cmd.Payload.Offer == "pilot-inputs-v2" {
			purchaseCost += packagingCost
			packs = breadCount
		}
		if cmd.Payload.Limit < purchaseCost || c.Cash < purchaseCost {
			return nil, "PRICE_OR_FUNDS"
		}
		c.Cash -= purchaseCost
		if c.CompletedBatches == 0 && c.GrainReady != 0 {
			c.CompletedBatches = 1 // Existing saves predate the batch counter.
		}
		c.BreadReady = 0
		c.OpeningFlourValue = c.FlourValue
		c.MillPurchaseCost = 0
		c.PaperPack += packs
		c.PackOrder = packs > 0
		c.Grain, c.GrainReady = 10_000, tick+1
		c.Water, c.Salt, c.Yeast = waterUsed, saltUsed, yeastUsed
		return []event{
			movement("purchase", map[string]int64{"cash_kopecks": -purchaseCost}),
			movement("raw_delivery", map[string]int64{"grain_g": 10_000, "water_g": waterUsed, "salt_g": saltUsed, "yeast_g": yeastUsed, "paper_pack_count": packs}),
		}, ""
	case "offer_output":
		if c.Batch != nil {
			return c.sellChain(cmd.Payload, tick)
		}
		if cmd.Payload.Quantity != breadCount || cmd.Payload.Price != breadPrice {
			return nil, "LOT_UNAVAILABLE"
		}
		stock, stockKey := &c.Bread, "bread_count"
		cost := int64(grainCost + ingredientsCost + transportCost + lineCost)
		if c.MillPurchaseCost != 0 {
			cost += c.MillPurchaseCost - millPrice
		}
		if c.PackOrder {
			stock, stockKey = &c.PackedBread, "packed_bread_count"
			cost += packagingCost
		}
		if c.BreadReady == 0 || tick < c.BreadReady || *stock != breadCount ||
			cmd.Payload.Lot != c.BreadLot {
			return nil, "LOT_UNAVAILABLE"
		}
		*stock = 0
		c.BreadLot = ""
		revenue := int64(breadCount * breadPrice)
		c.recordSale(revenue, cost+c.OpeningFlourValue-c.FlourValue)
		return []event{movement("sale", map[string]int64{stockKey: -breadCount, "cash_kopecks": revenue})}, ""
	default:
		return nil, "INVALID_COMMAND"
	}
	return nil, ""
}

func produce(c *company, tick int64) []event {
	if !c.Line {
		return nil
	}
	if c.Batch != nil {
		return c.produceChain(tick)
	}
	if c.Grain > 0 && tick >= c.GrainReady && c.Cash >= transportCost {
		c.Cash -= transportCost
		c.Grain, c.FlourReady = 0, tick+1
		c.Flour += flourProduced
		c.FlourValue += grainCost + transportCost
		return []event{
			movement("milling", map[string]int64{"grain_g": -10_000, "mill_flour_g": flourProduced, "residue_g": 2_600}),
			movement("flour_delivery", map[string]int64{"cash_kopecks": -transportCost, "mill_flour_g": -flourProduced, "bakery_flour_g": flourProduced}),
		}
	}
	if c.Grain == 0 && c.Flour >= flourUsed && tick >= c.FlourReady && c.Water >= waterUsed &&
		c.Salt >= saltUsed && c.Yeast >= yeastUsed && c.Cash >= lineCost &&
		(!c.PackOrder || c.PaperPack >= breadCount) {
		c.Cash -= lineCost
		remaining := c.Flour - flourUsed
		c.FlourValue = (c.FlourValue*remaining + c.Flour/2) / c.Flour
		c.Flour = remaining
		c.Water -= waterUsed
		c.Salt -= saltUsed
		c.Yeast -= yeastUsed
		c.Bread, c.BreadReady = breadCount, tick+1
		c.BreadLot = "bread-" + strconv.FormatInt(tick, 10)
		events := []event{
			movement("mixing", map[string]int64{"bakery_flour_g": -flourUsed, "water_g": -waterUsed, "salt_g": -saltUsed, "yeast_g": -yeastUsed, "dough_g": 12_896}),
			movement("baking", map[string]int64{"dough_g": -12_896, "bread_count": breadCount, "bake_loss_g": 1_496, "cash_kopecks": -lineCost}),
		}
		if c.PackOrder {
			c.Bread -= breadCount
			c.PaperPack -= breadCount
			c.PackedBread += breadCount
			events = append(events, movement("packing", map[string]int64{
				"bread_count": -breadCount, "paper_pack_count": -breadCount, "packed_bread_count": breadCount,
			}))
		}
		return events
	}
	return nil
}

func (c *company) recordSale(revenue, cost int64) {
	c.CompletedBatches++
	c.Revenue += revenue
	c.Cost += cost
	c.Profit = c.Revenue - c.Cost
	c.Cash += revenue
}

func decodeCompany(raw []byte) (company, error) {
	var c company
	err := json.Unmarshal(raw, &c)
	return c, err
}

var (
	errConflict   = errors.New("COMMAND_CONFLICT")
	errRecovering = errors.New("WORLD_RECOVERING")
	errScope      = errors.New("NOT_FOUND")
	errTickOrder  = errors.New("TICK_OUT_OF_ORDER")
)
