package main

import "strconv"

type chainStage struct {
	name    string
	in, out map[string]int64
}

type pilotChain struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Cost     int64  `json:"cost,string"`
	Output   string `json:"output"`
	Quantity int64  `json:"quantity,string"`
	Price    int64  `json:"price,string"`
	input    map[string]int64
	stages   []chainStage
}

// Amounts are provisional game values. Weights are grams.
var pilotChains = []pilotChain{
	{"milk", "Молоко → сливки и обезжиренное молоко", 40000, "cream_g", 7900, 7,
		map[string]int64{"raw_milk_g": 79900}, []chainStage{
			{"separation", map[string]int64{"raw_milk_g": 79900}, map[string]int64{"cream_g": 7900, "skimmed_milk_g": 72000}},
		}},
	{"wood", "Брёвна → щепа → пеллеты", 20000, "wood_pellets_g", 7000, 4,
		map[string]int64{"logs_g": 10000}, []chainStage{
			{"chipping", map[string]int64{"logs_g": 10000}, map[string]int64{"wood_chips_g": 9500}},
			{"pelletizing", map[string]int64{"wood_chips_g": 9500}, map[string]int64{"wood_pellets_g": 7000}},
		}},
	{"glass", "Стекольная шихта → стекломасса → банки", 24000, "glass_jar_count", 24, 1400,
		map[string]int64{"quartz_sand_g": 7000, "limestone_g": 1500, "soda_ash_g": 2000}, []chainStage{
			{"glass_melting", map[string]int64{"quartz_sand_g": 7000, "limestone_g": 1500, "soda_ash_g": 2000}, map[string]int64{"glass_melt_g": 9000}},
			{"jar_forming", map[string]int64{"glass_melt_g": 9000}, map[string]int64{"glass_jar_count": 24}},
		}},
	{"vegetables", "Томаты → паста", 18000, "tomato_paste_g", 2000, 12,
		map[string]int64{"tomato_g": 12000}, []chainStage{
			{"paste_concentration", map[string]int64{"tomato_g": 12000}, map[string]int64{"tomato_paste_g": 2000}},
		}},
	{"construction", "Цемент и песок → сухая смесь → раствор", 25000, "mortar_mix_g", 12000, 3,
		map[string]int64{"cement_g": 2500, "sand_supply_g": 7500, "water_g": 2000}, []chainStage{
			{"dry_mortar_blending", map[string]int64{"cement_g": 2500, "sand_supply_g": 7500}, map[string]int64{"dry_mortar_g": 10000}},
			{"mortar_mixing", map[string]int64{"dry_mortar_g": 10000, "water_g": 2000}, map[string]int64{"mortar_mix_g": 12000}},
		}},
	{"oil", "Нефть → базовое масло → фасованное масло", 32000, "lubricant_pack_count", 24, 1800,
		map[string]int64{"crude_oil_g": 24000, "plastic_bottle_count": 24, "bottle_cap_count": 24}, []chainStage{
			{"base_oil_refining", map[string]int64{"crude_oil_g": 24000}, map[string]int64{"lubricant_base_g": 12000}},
			{"base_oil_packing", map[string]int64{"lubricant_base_g": 12000, "plastic_bottle_count": 24, "bottle_cap_count": 24}, map[string]int64{"lubricant_pack_count": 24}},
		}},
}

type advanceOrder struct {
	Delivery int64 `json:"delivery_tick,string"`
	Price    int64 `json:"price_kopecks,string"`
}

type chainBatch struct {
	Next         *advanceOrder    `json:"next_order,omitempty"`
	Delivery     int64            `json:"delivery_tick,string"`
	PurchaseCost int64            `json:"purchase_cost_kopecks,string"`
	ID           string           `json:"id"`
	Stock        map[string]int64 `json:"stock"`
	Stage        int              `json:"stage"`
	Ready        int64            `json:"ready_tick,string"`
}

func chainByID(id string) *pilotChain {
	for i := range pilotChains {
		if pilotChains[i].ID == id {
			return &pilotChains[i]
		}
	}
	return nil
}

func (c *company) buyChain(p *pilotChain, limit, tick int64) ([]event, string) {
	if c.Cash < p.Cost || limit < p.Cost {
		return nil, "PRICE_OR_FUNDS"
	}
	c.startChain(p, p.Cost, tick)
	return append([]event{movement("purchase", map[string]int64{"cash_kopecks": -p.Cost})}, c.deliverChain(tick)...), ""
}

func (c *company) startChain(p *pilotChain, cost, delivery int64) {
	c.Cash -= cost
	if c.Batch == nil {
		c.Batch = &chainBatch{ID: p.ID, Stock: make(map[string]int64, len(p.input))}
	} else if c.CompletedBatches == 0 {
		c.CompletedBatches = 1 // Existing saves predate the batch counter.
	}
	c.Batch.Ready, c.Batch.Stage = delivery+1, 0
	c.Batch.Delivery, c.Batch.PurchaseCost = delivery, cost
	c.BreadReady = 0
}

// Supply for the next batch is held separately until the current sale closes.
func (c *company) advanceChainOrder(tick int64) []event {
	batch := c.Batch
	if batch == nil || batch.Next == nil || c.batchActive() {
		return nil
	}
	next := batch.Next
	batch.Next = nil
	batch.Delivery, batch.PurchaseCost = next.Delivery, next.Price
	batch.Ready, batch.Stage = next.Delivery+1, 0
	c.BreadReady = 0
	return c.deliverChain(tick)
}

func (c *company) produceChain(tick int64) []event {
	batch := c.Batch
	p := chainByID(batch.ID)
	if p == nil || batch.Stage >= len(p.stages) || tick < batch.Ready {
		return nil
	}
	stage := p.stages[batch.Stage]
	for k, v := range stage.in {
		if batch.Stock[k] < v {
			return nil
		}
	}
	delta := make(map[string]int64, len(stage.in)+len(stage.out))
	for k, v := range stage.in {
		delta[k] -= v
	}
	for k, v := range stage.out {
		delta[k] += v
	}
	for k, v := range delta {
		batch.Stock[k] += v
	}
	batch.Stage++
	batch.Ready = tick + 1
	if batch.Stage == len(p.stages) {
		c.BreadReady = batch.Ready
		c.BreadLot = "bread-" + strconv.FormatInt(tick, 10)
	}
	return []event{movement(stage.name, delta)}
}

func (c *company) sellChain(payload payload, tick int64) ([]event, string) {
	p := chainByID(c.Batch.ID)
	if p == nil || c.BreadLot == "" || payload.Lot != c.BreadLot || tick < c.BreadReady ||
		payload.Price != p.Price || payload.Quantity != p.Quantity || c.Batch.Stock[p.Output] < p.Quantity {
		return nil, "LOT_UNAVAILABLE"
	}
	c.Batch.Stock[p.Output] -= p.Quantity
	c.BreadLot = ""
	revenue := p.Quantity * p.Price
	cost := c.Batch.PurchaseCost
	if cost == 0 {
		cost = p.Cost
	}
	c.recordSale(revenue, cost)
	return []event{movement("sale", map[string]int64{p.Output: -p.Quantity, "cash_kopecks": revenue})}, ""
}
