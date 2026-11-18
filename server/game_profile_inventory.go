package main

import "fmt"

// Isolated-world inventory lots. The two timestamps deliberately differ:
// delivery may enter the warehouse at T but cannot be produced with until T+1;
// production at T cannot be sold until T+1.
type gameInventoryLot struct {
	Good                       string `json:"good"`
	Quantity                   int64  `json:"quantity"`
	AvailableForProductionTick int64  `json:"available_for_production_tick"`
	AvailableForSaleTick       int64  `json:"available_for_sale_tick"`
}

type gameInventory struct {
	Lots     []gameInventoryLot `json:"lots"`
	Reserved map[string]int64   `json:"reserved,omitempty"`
}

func (inv gameInventory) available(good string, tick int64, forSale bool) (int64, error) {
	var total int64
	for _, lot := range inv.Lots {
		if lot.Good == "" || lot.Quantity <= 0 || lot.AvailableForProductionTick < 0 || lot.AvailableForSaleTick < 0 {
			return 0, fmt.Errorf("INVALID_INVENTORY_LOT")
		}
		if lot.Good != good {
			continue
		}
		ready := lot.AvailableForProductionTick
		if forSale {
			ready = lot.AvailableForSaleTick
		}
		if ready > tick {
			continue
		}
		next, err := safeGameAdd(total, lot.Quantity)
		if err != nil {
			return 0, err
		}
		total = next
	}
	if inv.Reserved[good] < 0 || inv.Reserved[good] > total {
		return 0, fmt.Errorf("INVALID_INVENTORY_RESERVATION")
	}
	return total - inv.Reserved[good], nil
}

func (inv *gameInventory) receiveDelivery(good string, quantity, tick int64) error {
	if good == "" || quantity <= 0 || tick < 0 || tick == int64(^uint64(0)>>1) {
		return fmt.Errorf("INVALID_DELIVERY")
	}
	// Validate before appending; caller must persist with the enclosing transaction.
	if _, err := safeGameAdd(tick, 1); err != nil {
		return err
	}
	inv.Lots = append(inv.Lots, gameInventoryLot{Good: good, Quantity: quantity, AvailableForProductionTick: tick + 1, AvailableForSaleTick: tick + 1})
	return nil
}

func (inv *gameInventory) receiveProduction(good string, quantity, tick int64) error {
	if good == "" || quantity <= 0 || tick < 0 {
		return fmt.Errorf("INVALID_PRODUCTION_LOT")
	}
	next, err := safeGameAdd(tick, 1)
	if err != nil {
		return err
	}
	inv.Lots = append(inv.Lots, gameInventoryLot{Good: good, Quantity: quantity, AvailableForProductionTick: tick, AvailableForSaleTick: next})
	return nil
}

func (inv *gameInventory) reserve(good string, quantity, tick int64) error {
	if good == "" || quantity <= 0 {
		return fmt.Errorf("INVALID_RESERVATION")
	}
	available, err := inv.available(good, tick, true)
	if err != nil {
		return err
	}
	if quantity > available {
		return fmt.Errorf("INSUFFICIENT_SALEABLE_STOCK")
	}
	next, err := safeGameAdd(inv.Reserved[good], quantity)
	if err != nil {
		return err
	}
	if inv.Reserved == nil {
		inv.Reserved = map[string]int64{}
	}
	inv.Reserved[good] = next
	return nil
}

func (inv *gameInventory) cancelReservation(good string, quantity int64) error {
	if good == "" || quantity <= 0 || quantity > inv.Reserved[good] {
		return fmt.Errorf("INVALID_RESERVATION_RELEASE")
	}
	inv.Reserved[good] -= quantity
	return nil
}

// shipReserved consumes only mature, reserved stock and updates a private copy.
// An error leaves all lots and reservations unchanged.
func (inv *gameInventory) shipReserved(good string, quantity, tick int64) error {
	if good == "" || quantity <= 0 || quantity > inv.Reserved[good] {
		return fmt.Errorf("UNRESERVED_SHIPMENT")
	}
	next := append([]gameInventoryLot(nil), inv.Lots...)
	remaining := quantity
	for i := range next {
		if next[i].Good != good || next[i].AvailableForSaleTick > tick {
			continue
		}
		take := next[i].Quantity
		if take > remaining {
			take = remaining
		}
		next[i].Quantity -= take
		remaining -= take
		if remaining == 0 {
			break
		}
	}
	if remaining != 0 {
		return fmt.Errorf("IMMATURE_SHIPMENT")
	}
	kept := make([]gameInventoryLot, 0, len(next))
	for _, lot := range next {
		if lot.Quantity > 0 {
			kept = append(kept, lot)
		}
	}
	inv.Lots = kept
	inv.Reserved[good] -= quantity
	return nil
}

// consumeForProduction removes only mature, unreserved lots. Consumption is
// atomic and preserves lots not yet delivered or reserved for customer orders.
func (inv *gameInventory) consumeForProduction(good string, quantity, tick int64) error {
	if good == "" || quantity <= 0 || tick < 0 {
		return fmt.Errorf("INVALID_PRODUCTION_CONSUMPTION")
	}
	available, err := inv.available(good, tick, false)
	if err != nil {
		return err
	}
	if quantity > available {
		return fmt.Errorf("INSUFFICIENT_PRODUCTION_STOCK")
	}
	next := append([]gameInventoryLot(nil), inv.Lots...)
	// Reserve protected stock from the oldest eligible lots first. Only the
	// unreserved portion can be consumed by production.
	remaining := quantity
	protected := inv.Reserved[good]
	for i := range next {
		if next[i].Good != good || next[i].AvailableForProductionTick > tick {
			continue
		}
		locked := int64(0)
		if next[i].AvailableForSaleTick <= tick {
			locked = next[i].Quantity
			if locked > protected {
				locked = protected
			}
			protected -= locked
		}
		free := next[i].Quantity - locked
		take := free
		if take > remaining {
			take = remaining
		}
		next[i].Quantity -= take
		remaining -= take
		if remaining == 0 {
			break
		}
	}
	if remaining != 0 {
		return fmt.Errorf("PRODUCTION_RESERVATION_CONFLICT")
	}
	kept := make([]gameInventoryLot, 0, len(next))
	for _, lot := range next {
		if lot.Quantity > 0 {
			kept = append(kept, lot)
		}
	}
	inv.Lots = kept
	return nil
}
