package main

import "fmt"

// applyGameInventoryRuns replays an already validated production plan against
// private inventory lots. It never mutates the original inventory on failure.
// The caller persists inventory, stock and cash in the same company transaction.
func applyGameInventoryRuns(set gameProfileSet, runs []gameRun, inventory gameInventory, tick int64) (gameInventory, error) {
	next := gameInventory{Lots: append([]gameInventoryLot(nil), inventory.Lots...), Reserved: make(map[string]int64, len(inventory.Reserved))}
	for k, v := range inventory.Reserved {
		next.Reserved[k] = v
	}
	for _, run := range runs {
		if run.Batches == 0 {
			continue
		}
		p := profileByID(set, run.ProfileID)
		if p.ID == "" {
			return inventory, fmt.Errorf("INVENTORY_PROFILE_MISSING")
		}
		for good, perBatch := range p.BatchInputs {
			quantity, err := safeGameMul(perBatch, run.Batches)
			if err != nil {
				return inventory, err
			}
			if err = next.consumeForProduction(good, quantity, tick); err != nil {
				return inventory, fmt.Errorf("INVENTORY_INPUT_%s: %w", good, err)
			}
		}
		for good, perBatch := range p.BatchOutputs {
			quantity, err := safeGameMul(perBatch, run.Batches)
			if err != nil {
				return inventory, err
			}
			if err = next.receiveProduction(good, quantity, tick); err != nil {
				return inventory, err
			}
		}
	}
	return next, nil
}

func gameInventoryTotals(inv gameInventory) (map[string]int64, error) {
	totals := map[string]int64{}
	for _, lot := range inv.Lots {
		if lot.Good == "" || lot.Quantity <= 0 || lot.AvailableForProductionTick < 0 || lot.AvailableForSaleTick < 0 {
			return nil, fmt.Errorf("INVALID_INVENTORY_LOT")
		}
		n, err := safeGameAdd(totals[lot.Good], lot.Quantity)
		if err != nil {
			return nil, err
		}
		totals[lot.Good] = n
	}
	for good, qty := range inv.Reserved {
		if good == "" || qty < 0 || qty > totals[good] {
			return nil, fmt.Errorf("INVALID_INVENTORY_RESERVATION")
		}
	}
	return totals, nil
}

func checkGameInventoryStock(inv gameInventory, stock map[string]int64) error {
	totals, err := gameInventoryTotals(inv)
	if err != nil {
		return err
	}
	for good, qty := range stock {
		if qty < 0 || totals[good] != qty {
			return fmt.Errorf("INVENTORY_STOCK_MISMATCH: %s", good)
		}
	}
	for good, qty := range totals {
		if stock[good] != qty {
			return fmt.Errorf("INVENTORY_STOCK_MISMATCH: %s", good)
		}
	}
	return nil
}
