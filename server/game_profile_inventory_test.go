package main

import "testing"

func TestGameInventoryDeliveryProductionSaleTPlusOne(t *testing.T) {
	var inv gameInventory
	if err := inv.receiveDelivery("grain", 100, 5); err != nil {
		t.Fatal(err)
	}
	if n, err := inv.available("grain", 5, false); err != nil || n != 0 {
		t.Fatalf("delivery usable too soon: %d %v", n, err)
	}
	if n, err := inv.available("grain", 6, false); err != nil || n != 100 {
		t.Fatalf("delivery not ready T+1: %d %v", n, err)
	}
	if err := inv.receiveProduction("bread", 24, 6); err != nil {
		t.Fatal(err)
	}
	if n, err := inv.available("bread", 6, false); err != nil || n != 24 {
		t.Fatalf("production not usable within tick: %d %v", n, err)
	}
	if n, err := inv.available("bread", 6, true); err != nil || n != 0 {
		t.Fatalf("fresh production sold too early: %d %v", n, err)
	}
	if err := inv.reserve("bread", 10, 6); err == nil {
		t.Fatal("fresh bread reserved for sale")
	}
	if err := inv.reserve("bread", 10, 7); err != nil {
		t.Fatal(err)
	}
	if n, err := inv.available("bread", 7, true); err != nil || n != 14 {
		t.Fatalf("reservation failed: %d %v", n, err)
	}
	if err := inv.shipReserved("bread", 11, 7); err == nil {
		t.Fatal("unreserved shipment")
	}
	if err := inv.shipReserved("bread", 6, 7); err != nil {
		t.Fatal(err)
	}
	if err := inv.cancelReservation("bread", 4); err != nil {
		t.Fatal(err)
	}
	if n, err := inv.available("bread", 7, true); err != nil || n != 18 {
		t.Fatalf("shipment/cancellation failed: %d %v", n, err)
	}
}

func TestGameInventoryShipmentAtomicOnImmatureLot(t *testing.T) {
	inv := gameInventory{Lots: []gameInventoryLot{{Good: "bread", Quantity: 4, AvailableForProductionTick: 1, AvailableForSaleTick: 10}}, Reserved: map[string]int64{"bread": 4}}
	if err := inv.shipReserved("bread", 4, 9); err == nil {
		t.Fatal("immature lot shipped")
	}
	if inv.Lots[0].Quantity != 4 || inv.Reserved["bread"] != 4 {
		t.Fatal("failed shipment mutated inventory")
	}
}

func TestGameInventoryProductionRespectsDeliveryAndReservation(t *testing.T) {
	var inv gameInventory
	if err := inv.receiveDelivery("grain", 100, 3); err != nil {
		t.Fatal(err)
	}
	if err := inv.consumeForProduction("grain", 1, 3); err == nil {
		t.Fatal("used delivery on confirmation tick")
	}
	if err := inv.consumeForProduction("grain", 40, 4); err != nil {
		t.Fatal(err)
	}
	if n, err := inv.available("grain", 4, false); err != nil || n != 60 {
		t.Fatalf("wrong grain balance %d %v", n, err)
	}
	if err := inv.reserve("grain", 50, 4); err != nil {
		t.Fatal(err)
	}
	if err := inv.consumeForProduction("grain", 11, 4); err == nil {
		t.Fatal("spent reserved grain")
	}
	if err := inv.consumeForProduction("grain", 10, 4); err != nil {
		t.Fatal(err)
	}
	if err := inv.shipReserved("grain", 50, 4); err != nil {
		t.Fatal(err)
	}
	if n, err := inv.available("grain", 4, false); err != nil || n != 0 {
		t.Fatalf("stock minted: %d %v", n, err)
	}
}
