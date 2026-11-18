package main

import "fmt"

const gameBreadLot = "game-bread-v1"

type gameFlourDelivery struct {
	Offer string `json:"offer_id"`
	Tick  int64  `json:"delivery_tick,string"`
	Price int64  `json:"price_kopecks,string"`
}

func gameFlourInputs() map[string]int64 {
	return map[string]int64{"wheat_flour": flourProduced, "water": waterUsed, "salt": saltUsed, "yeast": yeastUsed}
}

func addGameFlourDelivery(inv *gameInventory, tick int64) (map[string]int64, error) {
	for good, qty := range gameFlourInputs() {
		if err := inv.receiveDelivery(good, qty, tick); err != nil {
			return nil, err
		}
	}
	return gameInventoryTotals(*inv)
}

func (c *company) arriveGameFlour(tick int64) ([]event, error) {
	p := c.GameProduction
	if p == nil || p.FlourDelivery == nil || p.FlourDelivery.Tick > tick {
		return nil, nil
	}
	inv, ok := prepareGameTrade(c)
	if !ok {
		return nil, fmt.Errorf("GAME_INVENTORY_REQUIRED")
	}
	totals, err := addGameFlourDelivery(&inv, tick)
	if err != nil {
		return nil, err
	}
	p.Inventory, p.Stock, p.FlourDelivery = &inv, totals, nil
	return []event{movement("game_flour_delivery", gameFlourInputs())}, nil
}

func prepareGameTrade(c *company) (gameInventory, bool) {
	p := c.GameProduction
	if p == nil || p.Inventory == nil || checkGameInventoryStock(*p.Inventory, p.Stock) != nil {
		return gameInventory{}, false
	}
	return copyGameInventory(*p.Inventory), true
}

func buyGameFlour(buyer, seller *company, p payload, tick int64) ([]event, []event, string) {
	inv, ok := prepareGameTrade(buyer)
	if !ok {
		return nil, nil, "GAME_INVENTORY_REQUIRED"
	}
	if seller.MillFlour < flourProduced || seller.MillUnlisted || p.Offer != seller.flourOfferID() {
		return nil, nil, "MILL_OFFER_UNAVAILABLE"
	}
	if buyer.GameProduction.FlourDelivery != nil {
		return nil, nil, "GAME_DELIVERY_PENDING"
	}
	price, delay := millPriceAndDelay(millSeller(p.Offer))
	cost := price + ingredientsCost
	if p.Limit < price || buyer.Cash < cost {
		return nil, nil, "PRICE_OR_FUNDS"
	}
	cash, err := safeGameAdd(seller.Cash, price)
	if err != nil {
		return nil, nil, "GAME_TRADE_OVERFLOW"
	}
	revenue, err := safeGameAdd(seller.Revenue, price)
	if err != nil {
		return nil, nil, "GAME_TRADE_OVERFLOW"
	}
	saleCost, err := safeGameAdd(seller.Cost, seller.MillCost)
	if err != nil {
		return nil, nil, "GAME_TRADE_OVERFLOW"
	}
	due, err := safeGameAdd(tick, delay)
	if err != nil {
		return nil, nil, "GAME_DELIVERY_INVALID"
	}
	totals, err := addGameFlourDelivery(&inv, due)
	if err != nil {
		return nil, nil, "GAME_DELIVERY_INVALID"
	}
	buyer.Cash -= cost
	events := []event{movement("game_flour_purchase", map[string]int64{"cash_kopecks": -cost})}
	if delay == 0 {
		buyer.GameProduction.Inventory, buyer.GameProduction.Stock = &inv, totals
		events = append(events, movement("game_flour_delivery", gameFlourInputs()))
	} else {
		buyer.GameProduction.FlourDelivery = &gameFlourDelivery{Offer: p.Offer, Tick: due, Price: price}
	}
	seller.Cash, seller.Revenue, seller.Cost = cash, revenue, saleCost
	seller.Profit = revenue - saleCost
	seller.MillFlour -= flourProduced
	seller.MillCost = 0
	return events,
		[]event{movement("mill_sale", map[string]int64{"mill_flour_g": -flourProduced, "cash_kopecks": price})}, ""
}

func (c *company) sellGameBread(tick int64) ([]event, string) {
	inv, ok := prepareGameTrade(c)
	if !ok {
		return nil, "GAME_INVENTORY_REQUIRED"
	}
	if err := inv.reserve("bread_loaf", breadCount, tick); err != nil {
		return nil, "LOT_UNAVAILABLE"
	}
	if err := inv.shipReserved("bread_loaf", breadCount, tick); err != nil {
		return nil, "LOT_UNAVAILABLE"
	}
	cash, err := safeGameAdd(c.Cash, breadCount*breadPrice)
	if err != nil {
		return nil, "GAME_TRADE_OVERFLOW"
	}
	revenue, err := safeGameAdd(c.Revenue, breadCount*breadPrice)
	if err != nil {
		return nil, "GAME_TRADE_OVERFLOW"
	}
	totals, err := gameInventoryTotals(inv)
	if err != nil {
		return nil, "GAME_SHIPMENT_INVALID"
	}
	c.Cash, c.Revenue = cash, revenue
	c.CompletedBatches++
	c.GameProduction.Inventory, c.GameProduction.Stock = &inv, totals
	return []event{movement("game_bread_sale", map[string]int64{"bread_loaf": -breadCount, "cash_kopecks": breadCount * breadPrice})}, ""
}
