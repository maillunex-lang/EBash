package main

type bakeryConstruction struct {
	Sites     map[string]int64 `json:"sites"`
	Duration  int64            `json:"duration"`
	Site      string           `json:"site"`
	ReadyTick int64            `json:"ready_tick"`
	Open      bool             `json:"open"`
}

func newBakeryConstruction() *bakeryConstruction {
	return &bakeryConstruction{Sites: map[string]int64{"6": 30000, "12": 40000, "18": 35000}, Duration: 2}
}

func startBakery(c *company, companies map[string]company, p payload, tick int64) ([]event, string) {
	if c.GameProduction == nil || c.GameProduction.Construction == nil {
		return nil, "CONSTRUCTION_DISABLED"
	}
	b := c.GameProduction.Construction
	price, ok := b.Sites[p.Offer]
	if !ok || b.Site != "" || price != p.Limit {
		return nil, "SITE_UNAVAILABLE"
	}
	for _, other := range companies {
		if other.GameProduction != nil && other.GameProduction.Construction != nil && other.GameProduction.Construction.Site == p.Offer {
			return nil, "SITE_UNAVAILABLE"
		}
	}
	if c.Cash < price {
		return nil, "INSUFFICIENT_FUNDS"
	}
	c.Cash -= price
	b.Site = p.Offer
	b.ReadyTick = tick + b.Duration
	return []event{movement("bakery_construction", map[string]int64{"cash_kopecks": -price})}, ""
}
