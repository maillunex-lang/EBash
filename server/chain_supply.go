package main

import (
	"strconv"
	"strings"
)

const supplyRestockDelay = 4

type chainSupply struct {
	ID          string `json:"offer_id"`
	Chain       string `json:"chain"`
	Label       string `json:"label"`
	Price       int64  `json:"price_kopecks,string"`
	Delay       int64  `json:"delivery_ticks"`
	RestockTick int64  `json:"restock_tick,string"`
	Available   bool   `json:"available"`
}

var chainSupplies = []chainSupply{
	{ID: "milk-near", Chain: "milk", Label: "Ближняя ферма", Price: 40000},
	{ID: "milk-remote", Chain: "milk", Label: "Дальняя ферма", Price: 34000, Delay: 2},
	{ID: "wood-near", Chain: "wood", Label: "Ближний лесной склад", Price: 20000},
	{ID: "wood-remote", Chain: "wood", Label: "Дальний лесной склад", Price: 17000, Delay: 2},
	{ID: "glass-near", Chain: "glass", Label: "Ближний склад стекольной шихты", Price: 24000},
	{ID: "glass-remote", Chain: "glass", Label: "Дальний склад стекольной шихты", Price: 20400, Delay: 2},
	{ID: "vegetables-near", Chain: "vegetables", Label: "Ближнее хозяйство", Price: 18000},
	{ID: "vegetables-remote", Chain: "vegetables", Label: "Дальнее хозяйство", Price: 15300, Delay: 2},
	{ID: "construction-near", Chain: "construction", Label: "Ближний склад стройсырья", Price: 25000},
	{ID: "construction-remote", Chain: "construction", Label: "Дальний склад стройсырья", Price: 21250, Delay: 2},
	{ID: "oil-near", Chain: "oil", Label: "Ближний склад нефти и тары", Price: 32000},
	{ID: "oil-remote", Chain: "oil", Label: "Дальний склад нефти и тары", Price: 27200, Delay: 2},
}

func supplyByID(id string) *chainSupply {
	base, suffix, versioned := strings.Cut(id, "@")
	if versioned {
		tick, err := strconv.ParseInt(suffix, 10, 64)
		if err != nil || tick <= 0 || strconv.FormatInt(tick, 10) != suffix {
			return nil
		}
	}
	id = base
	for i := range chainSupplies {
		if chainSupplies[i].ID == id {
			return &chainSupplies[i]
		}
	}
	return nil
}

func (s chainSupply) account() string { return "supplier-" + s.ID }

func accountSupply(account string) *chainSupply {
	for i := range chainSupplies {
		if chainSupplies[i].account() == account {
			return &chainSupplies[i]
		}
	}
	return nil
}

func (s chainSupply) lotID(seller company) string {
	if seller.SupplyLotTick == 0 {
		return s.ID
	}
	return s.ID + "@" + strconv.FormatInt(seller.SupplyLotTick, 10)
}

func (c *company) restockSupply(offer *chainSupply, tick int64) []event {
	if c.SupplyLot {
		return nil
	}
	if c.SupplyRestockTick == 0 {
		c.SupplyRestockTick = tick + supplyRestockDelay
	}
	cost := offer.Price * 3 / 4
	if tick < c.SupplyRestockTick || c.Cash < cost {
		return nil
	}
	c.Cash -= cost
	c.SupplyCost = cost
	c.SupplyLot = true
	c.SupplyLotTick = tick
	c.SupplyRestockTick = 0
	return []event{movement("supplier_restock", map[string]int64{"cash_kopecks": -cost, "supply_lots": 1})}
}

func buyChainSupply(buyer, seller *company, offer *chainSupply, lot string, limit, tick int64) ([]event, []event, string) {
	if buyer.GrainReady != 0 || (buyer.Batch != nil && buyer.Batch.ID != offer.Chain) {
		return nil, nil, "PILOT_CHAIN_LOCKED"
	}
	if buyer.Batch != nil && buyer.Batch.Next != nil {
		return nil, nil, "ADVANCE_ORDER_EXISTS"
	}
	if !seller.SupplyLot || lot != offer.lotID(*seller) {
		return nil, nil, "SUPPLY_OFFER_UNAVAILABLE"
	}
	if buyer.Cash < offer.Price || limit < offer.Price {
		return nil, nil, "PRICE_OR_FUNDS"
	}
	chain := chainByID(offer.Chain)
	var delivery []event
	if buyer.batchActive() {
		buyer.Cash -= offer.Price
		buyer.Batch.Next = &advanceOrder{Delivery: tick + offer.Delay, Price: offer.Price}
	} else {
		buyer.startChain(chain, offer.Price, tick+offer.Delay)
		delivery = buyer.deliverChain(tick)
	}
	seller.SupplyLot = false
	seller.SupplyRestockTick = tick + supplyRestockDelay
	seller.Cash += offer.Price
	seller.Revenue += offer.Price
	seller.Cost += seller.SupplyCost
	seller.Profit = seller.Revenue - seller.Cost
	seller.SupplyCost = 0
	return append([]event{movement("supplier_purchase", map[string]int64{"cash_kopecks": -offer.Price})}, delivery...),
		[]event{movement("supplier_sale", map[string]int64{"cash_kopecks": offer.Price, "supply_lots": -1})}, ""
}

func (c *company) deliverChain(tick int64) []event {
	batch := c.Batch
	if batch == nil || batch.Delivery == 0 || tick < batch.Delivery {
		return nil
	}
	batch.Delivery = 0
	input := chainByID(batch.ID).input
	for k, v := range input {
		batch.Stock[k] += v
	}
	return []event{movement("raw_delivery", input)}
}
