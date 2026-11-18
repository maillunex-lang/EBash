package main

import (
	"strconv"
	"strings"
)

const (
	remoteMill         = "pilot-mill-remote"
	remoteMillPrice    = 36000
	remoteDelay        = 2
	millAccount        = "pilot-mill"
	millOfferID        = "pilot-mill-flour-v1"
	millPrice          = grainCost + transportCost
	bakerySuppliesCost = ingredientsCost + packagingCost
)

type millOffer struct {
	Delay     int    `json:"delivery_ticks"`
	ID        string `json:"offer_id"`
	Seller    bool   `json:"seller"`
	Available bool   `json:"available"`
	Quantity  int64  `json:"quantity_g,string"`
	Price     int64  `json:"price_kopecks,string"`
	Supplies  int64  `json:"ingredients_kopecks,string"`
}

func isMill(account string) bool { return account == millAccount || account == remoteMill }

func millSeller(offer string) string {
	if strings.HasPrefix(offer, "remote-") {
		return remoteMill
	}
	return millAccount
}

func (c company) flourOfferID() string {
	if c.MillLot == "" {
		return millOfferID
	}
	return c.MillLot
}

func validMillLot(id string) bool {
	id = strings.TrimPrefix(id, "remote-")
	if id == millOfferID {
		return true
	}
	suffix, ok := strings.CutPrefix(id, "pilot-mill-flour-")
	tick, err := strconv.ParseInt(suffix, 10, 64)
	return ok && err == nil && tick > 0
}

func (c *company) applyMillCommand(account, kind string, tick int64) ([]event, string) {
	switch kind {
	case "buy_mill_grain":
		if c.MillGrain != 0 || c.MillFlour != 0 {
			return nil, "MILL_STOCK_EXISTS"
		}
		if c.Cash < grainCost {
			return nil, "PRICE_OR_FUNDS"
		}
		c.Cash -= grainCost
		c.MillGrain, c.MillGrainReady = 10000, tick+1
		return []event{movement("mill_grain_purchase", map[string]int64{"cash_kopecks": -grainCost, "mill_grain_g": 10000})}, ""
	case "list_mill_flour":
		if !c.MillUnlisted || c.MillFlour != flourProduced {
			return nil, "MILL_FLOUR_NOT_READY"
		}
		c.MillUnlisted = false
		c.MillLot = "pilot-mill-flour-" + strconv.FormatInt(tick, 10)
		if account == remoteMill {
			c.MillLot = "remote-" + c.MillLot
		}
		return []event{movement("mill_flour_listed", nil)}, ""
	}
	return nil, "INVALID_COMMAND"
}

func (c *company) produceMill(tick int64) []event {
	if c.MillGrain != 10000 || tick < c.MillGrainReady {
		return nil
	}
	c.MillGrain = 0
	c.MillFlour = flourProduced
	c.MillCost = grainCost
	c.MillUnlisted = true
	return []event{movement("mill_production", map[string]int64{"mill_grain_g": -10000, "mill_flour_g": flourProduced, "residue_g": 10000 - flourProduced})}
}

func buyMillFlour(buyer, seller *company, account, offer string, limit, tick int64) ([]event, []event, string) {
	price, delay := int64(millPrice), int64(0)
	if millSeller(offer) == remoteMill {
		price, delay = remoteMillPrice, remoteDelay
	}
	switch {
	case isMill(account) || seller.MillFlour < flourProduced || seller.MillUnlisted || offer != seller.flourOfferID():
		return nil, nil, "MILL_OFFER_UNAVAILABLE"
	case buyer.NextMill != nil:
		return nil, nil, "ADVANCE_ORDER_EXISTS"
	case buyer.Batch != nil:
		return nil, nil, "PILOT_CHAIN_LOCKED"
	case limit < price || buyer.Cash < price+bakerySuppliesCost:
		return nil, nil, "PRICE_OR_FUNDS"
	}
	seller.MillFlour -= flourProduced
	seller.Cash += price
	seller.Revenue += price
	seller.Cost += seller.MillCost
	seller.Profit = seller.Revenue - seller.Cost
	seller.MillCost = 0

	buyer.Cash -= price + bakerySuppliesCost
	next := advanceOrder{Delivery: tick + delay, Price: price}
	events := []event{movement("mill_purchase", map[string]int64{"cash_kopecks": -price, "transit_flour_g": flourProduced})}
	if buyer.batchActive() {
		buyer.NextMill = &next
		events = append(events, movement("bakery_reserve_purchase", map[string]int64{"cash_kopecks": -bakerySuppliesCost, "reserved_bakery_supplies": 1}))
	} else {
		events = append(events, movement("bakery_supplies_purchase", map[string]int64{"cash_kopecks": -bakerySuppliesCost}))
		events = append(events, buyer.startMillOrder(next, tick)...)
	}
	return events, []event{movement("mill_sale", map[string]int64{"cash_kopecks": price, "mill_flour_g": -flourProduced})}, ""
}

func (c *company) startMillOrder(order advanceOrder, tick int64) []event {
	if c.CompletedBatches == 0 && c.GrainReady != 0 {
		c.CompletedBatches = 1
	}
	c.OpeningFlourValue = c.FlourValue
	c.GrainReady = tick + 1
	c.BreadReady = 0
	c.MillPurchaseCost = order.Price
	c.FlourReady = order.Delivery + 1
	c.InTransitUntil = order.Delivery
	c.Water, c.Salt, c.Yeast = waterUsed, saltUsed, yeastUsed
	c.PaperPack += breadCount
	c.PackOrder = true
	return append([]event{movement("bakery_supplies", map[string]int64{"water_g": waterUsed, "salt_g": saltUsed, "yeast_g": yeastUsed, "paper_pack_count": breadCount})}, c.deliverMillFlour(tick)...)
}

func (c *company) advanceMillOrder(tick int64) []event {
	if c.NextMill == nil || c.batchActive() {
		return nil
	}
	next := *c.NextMill
	c.NextMill = nil
	return append([]event{movement("bakery_reserve_release", map[string]int64{"reserved_bakery_supplies": -1})}, c.startMillOrder(next, tick)...)
}

func (c *company) deliverMillFlour(tick int64) []event {
	if c.InTransitUntil == 0 || tick < c.InTransitUntil {
		return nil
	}
	c.InTransitUntil = 0
	c.Flour += flourProduced
	c.FlourValue += c.MillPurchaseCost
	return []event{movement("mill_delivery", map[string]int64{"transit_flour_g": -flourProduced, "bakery_flour_g": flourProduced})}
}
