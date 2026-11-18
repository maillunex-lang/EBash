extends SceneTree

const PilotPanel := preload("res://PilotPanel.gd")
const PilotClient := preload("res://PilotClient.gd")
const PilotScene := preload("res://Pilot.tscn")


func _initialize() -> void:
	call_deferred("run")


func run() -> void:
	# HTTP callbacks advance on frames. Keep this headless test independent of device preferences.
	Engine.max_fps = 0
	var panel := PilotScene.instantiate() as PilotPanel
	root.add_child(panel)
	while panel.busy:
		await process_frame
	assert(panel.fresh and panel.snapshot["cash_kopecks"] == "200000")
	assert(panel.sell_button.disabled)
	if panel.profile_demo:
		panel.queue_free()
		await process_frame
		assert(change_scene_to_file("res://Launcher.tscn") == OK)
		await scene_changed
		current_scene.play_button.pressed.emit()
		await scene_changed
		current_scene.city_buttons["ufa"].pressed.emit()
		await scene_changed
		select_city_supplier(current_scene, "near")
		current_scene.bakery_button.pressed.emit()
		await scene_changed
		panel = current_scene as PilotPanel
		while panel.busy:
			await process_frame
		await run_profile_demo(panel)
		return
	if panel.demo:
		await run_bakery_demo(panel)
		return
	assert(not panel.tick_button.visible)
	var mill_token := OS.get_environment("EBASH_PILOT_MILL_TOKEN")
	if not mill_token.is_empty():
		await run_mill_trade(panel, mill_token)
		return
	var chain_index := OS.get_environment("EBASH_PILOT_CHAIN_INDEX").to_int()
	if chain_index > 0:
		await run_chain(panel, chain_index)
		return
	await panel.order_inputs()
	assert(panel.order_state["status"] == "accepted" and panel.buy_button.disabled)
	await panel.configure_line()
	assert(panel.line_state["status"] == "accepted")
	run_tick(1)
	await panel.refresh()
	assert(panel.snapshot["grain_g"] == "10000" and panel.snapshot["paper_pack_count"] == "24")
	assert(panel.order_state["status"] == "applied" and panel.line_state["status"] == "applied")
	run_tick(2)
	await panel.refresh()
	assert(panel.snapshot["flour_g"] == "7400" and panel.sell_button.disabled)
	run_tick(3)
	await panel.refresh()
	assert(
		(
			panel.snapshot["packed_bread_count"] == "24"
			and panel.snapshot["paper_pack_count"] == "0"
			and not panel.sell_button.disabled
		)
	)
	await panel.sell_bread()
	assert(panel.sale_state["status"] == "accepted")
	run_tick(4)
	run_tick(4)
	await panel.refresh()
	assert(panel.sale_state["status"] == "applied" and panel.sell_button.disabled)
	assert(panel.snapshot["cash_kopecks"] == "231600")
	assert(panel.snapshot["profit_kopecks"] == "32735")
	assert(panel.snapshot["flour_g"] == "200" and panel.snapshot["packed_bread_count"] == "0")
	assert(panel.money("profit_kopecks") == "327,35")
	panel.queue_free()
	await process_frame
	var reopened := PilotScene.instantiate() as PilotPanel
	root.add_child(reopened)
	while reopened.busy:
		await process_frame
	assert(reopened.sale_state["status"] == "applied" and not reopened.buy_button.disabled)
	for batch in range(2, 4):
		await reopened.order_inputs()
		assert(reopened.order_state["status"] == "accepted")
		var duplicate := await reopened.api.submit(
			reopened.order_id(),
			"order_raw",
			{"offer_id": "pilot-inputs-v2", "quantity": "1", "max_price_kopecks": "50400"}
		)
		assert(duplicate["status"] == 202)
		var first_tick := 4 * (batch - 1) + 1
		run_tick(first_tick)
		run_tick(first_tick)
		run_tick(first_tick + 1)
		run_tick(first_tick + 2)
		await reopened.refresh()
		assert(reopened.snapshot["bread_lot_id"] == "bread-" + str(first_tick + 2))
		await reopened.sell_bread()
		run_tick(first_tick + 3)
		run_tick(first_tick + 3)
		await reopened.refresh()
		assert(int(reopened.snapshot["cash_kopecks"]) == 200000 + 31600 * batch)
		assert(int(reopened.snapshot["flour_g"]) == 200 * batch)
		assert(int(reopened.snapshot["completed_batches"]) == batch)
		assert(
			(
				int(reopened.snapshot["profit_kopecks"])
				== 31600 * batch + int(reopened.snapshot["flour_value_kopecks"])
			)
		)
		assert(not reopened.buy_button.disabled)
	print("PILOT SMOKE OK: buy, configure, four ticks, sale, repeat tick and reopen")
	quit()


func select_city_supplier(city: Node, kind: String) -> void:
	for i in range(city.district.lots.size()):
		var lot: Dictionary = city.district.lots[i]
		if lot.kind == kind:
			var center: Vector2 = lot.plot.get_center()
			var screen: Vector2 = city.camera.unproject_position(Vector3(center.x,lot.height,center.y))
			city.pick_lot(screen)
			assert(city.active_lot == i and city.bakery_button.visible)
			return
	assert(false, "Мельница отсутствует на карте")


func run_profile_demo(panel: PilotPanel) -> void:
	assert(panel.demo and panel.tick_button.visible and not panel.line_button.visible)
	assert(panel.screen_title.text == "Пекарня" and panel.bakery_goal.text.contains("0 из 2"))
	assert(panel.mill_buy_button.visible and not panel.mill_buy_button.disabled and not panel.remote_buy_button.visible)
	assert(panel.mill_offer_label.text.contains("600,00") and panel.remote_buy_button.text.contains("540,00"))
	await panel.order_mill_flour()
	await panel.advance_demo_tick()
	assert(int(panel.profile_stock()["wheat_flour"]) == 7400 and panel.sell_button.disabled)
	await panel.advance_demo_tick()
	assert(int(panel.profile_stock()["bread_loaf"]) == 24 and not panel.sell_button.disabled)
	await panel.sell_bread()
	await panel.advance_demo_tick()
	assert(panel.snapshot["completed_batches"] == "1" and panel.snapshot["cash_kopecks"] == "234000")
	panel.return_to_city()
	await scene_changed
	var city := current_scene
	while not city.bakery_status.text.contains("Продано партий: 1"):
		await process_frame
	select_city_supplier(city, "remote")
	city.bakery_button.pressed.emit()
	await scene_changed
	panel = current_scene as PilotPanel
	while panel.busy:
		await process_frame
	assert(panel.snapshot["cash_kopecks"] == "234000")

	assert(panel.remote_buy_button.visible and not panel.remote_buy_button.disabled and not panel.mill_buy_button.visible)
	await panel.order_remote_flour()
	await panel.advance_demo_tick()
	assert(panel.demo_hint.text.contains("Прибытие на тике 6"))
	assert(int(panel.profile_stock()["wheat_flour"]) == 200 and panel.sell_button.disabled)
	await panel.advance_demo_tick()
	assert(int(panel.profile_stock()["wheat_flour"]) == 200)
	await panel.advance_demo_tick()
	assert(int(panel.profile_stock()["wheat_flour"]) == 7600 and panel.sell_button.disabled)
	await panel.advance_demo_tick()
	assert(int(panel.profile_stock()["bread_loaf"]) == 24 and not panel.sell_button.disabled)
	await panel.sell_bread()
	await panel.advance_demo_tick()
	assert(panel.snapshot["completed_batches"] == "2" and panel.snapshot["cash_kopecks"] == "274000")
	assert(int(panel.profile_stock()["wheat_flour"]) == 400 and panel.demo_hint.text.contains("Две партии проданы"))
	assert(panel.bakery_goal.text.contains("2 из 2"))
	print("E104 DEMO SMOKE OK: two suppliers, delivered stock and sales")
	quit()


func run_bakery_demo(panel: PilotPanel) -> void:
	assert(panel.tick_button.visible and not panel.tick_button.disabled)
	assert(
		(
			not panel.buy_button.visible
			and panel.remote_buy_button.visible
			and not panel.mill_buy_button.disabled
		)
	)
	await panel.configure_line()
	await panel.order_mill_flour()
	await panel.advance_demo_tick()
	assert(panel.snapshot["last_committed_tick"] == "1" and panel.snapshot["flour_g"] == "7400")
	var duplicate := await panel.api.send(HTTPClient.METHOD_POST, "/demo-tick/1")
	assert(duplicate["status"] == 200)
	await panel.refresh()
	assert(panel.snapshot["last_committed_tick"] == "1")
	await panel.order_remote_flour()
	await panel.advance_demo_tick()
	assert(panel.snapshot["packed_bread_count"] == "24")
	assert(panel.snapshot["next_mill_order"]["delivery_tick"] == "4")
	await panel.sell_bread()
	await panel.advance_demo_tick()
	assert(panel.snapshot["completed_batches"] == "1")
	# Simulate an Android client being closed while a remote order is in transit.
	var pending_cash: String = str(panel.snapshot["cash_kopecks"])
	var pending_delivery: String = str(panel.snapshot["mill_delivery_tick"])
	panel.queue_free()
	await process_frame
	panel = PilotScene.instantiate() as PilotPanel
	root.add_child(panel)
	while panel.busy:
		await process_frame
	assert(panel.fresh and panel.snapshot["cash_kopecks"] == pending_cash)
	assert(panel.snapshot["mill_delivery_tick"] == pending_delivery)
	assert(panel.snapshot["mill_delivery_tick"] == "4")
	await panel.advance_demo_tick()
	assert(panel.snapshot["flour_g"] == "7600" and panel.sell_button.disabled)
	await panel.advance_demo_tick()
	assert(panel.snapshot["packed_bread_count"] == "24")
	await panel.sell_bread()
	await panel.advance_demo_tick()
	assert(
		panel.snapshot["completed_batches"] == "2" and panel.snapshot["cash_kopecks"] == "269200"
	)
	assert(panel.demo_hint.text.begins_with("Две партии проданы"))
	print("BAKERY DEMO SMOKE OK: two batches using only game controls")
	quit()


func run_tick(tick: int) -> void:
	var binary := OS.get_environment("EBASH_PILOT_SERVER_BIN")
	assert(not binary.is_empty())
	var args := PackedStringArray(
		["tick", "-world", OS.get_environment("EBASH_PILOT_WORLD"), "-tick", str(tick)]
	)
	assert(OS.execute(binary, args) == 0)


func run_chain(panel: PilotPanel, chain_index: int) -> void:
	assert(panel.chain_picker.item_count == 7)
	panel.chain_picker.select(chain_index)
	var profile := panel.selected_chain()
	await panel.configure_line()
	assert(panel.line_state["status"] == "accepted")
	var first_tick := 1
	var offers := panel.supplies.filter(
		func(offer: Dictionary) -> bool: return offer["chain"] == profile["id"]
	)
	assert(offers.size() == 2)
	assert(int(offers[1]["price_kopecks"]) < int(offers[0]["price_kopecks"]))
	var total_cost := 0
	var total_revenue := 0
	for batch in range(1, 4):
		var supply: Dictionary
		if batch < 3:
			assert(panel.supply_buttons.get_child_count() == 3 - batch)
			supply = offers[batch - 1]
		else:
			var refreshed := panel.supplies.filter(
				func(offer: Dictionary) -> bool: return offer["chain"] == profile["id"]
			)
			assert(refreshed.size() == 2)
			supply = refreshed[0]
			assert(str(supply["offer_id"]).begins_with(str(offers[0]["offer_id"]) + "@"))
			assert(supply["price_kopecks"] == offers[0]["price_kopecks"])
		assert(supply["available"])
		var delay := int(supply["delivery_ticks"])
		assert(delay == (2 if batch == 2 else 0))
		total_cost += int(supply["price_kopecks"])
		await panel.order_supply(supply)
		assert(panel.supply_states[supply["offer_id"]]["status"] == "accepted")
		run_tick(first_tick)
		await panel.refresh()
		assert(panel.snapshot["chain_batch"]["stage"] == 0 and panel.sell_button.disabled)
		if delay > 0:
			assert(int(panel.snapshot["chain_batch"]["delivery_tick"]) == first_tick + delay)
			for transit_tick in range(first_tick + 1, first_tick + delay + 1):
				run_tick(transit_tick)
				run_tick(transit_tick)
				await panel.refresh()
				assert(panel.snapshot["chain_batch"]["stage"] == 0 and panel.sell_button.disabled)
		var tick := first_tick + delay + 1
		while tick <= first_tick + delay + 2:
			run_tick(tick)
			await panel.refresh()
			if profile["id"] == "oil" and tick == first_tick + delay + 1:
				var intermediate: Dictionary = panel.snapshot["chain_batch"]["stock"]
				assert(
					(
						int(intermediate["crude_oil_g"]) == 0
						and int(intermediate["lubricant_base_g"]) == 12000
					)
				)
				assert(
					(
						int(intermediate["plastic_bottle_count"]) == 24
						and int(intermediate["bottle_cap_count"]) == 24
					)
				)
				assert(panel.sell_button.disabled)
			if not panel.sell_button.disabled:
				break
			tick += 1
		assert(not panel.sell_button.disabled)
		assert(
			(
				int(panel.snapshot["chain_batch"]["stock"][profile["output"]])
				== int(profile["quantity"])
			)
		)
		if profile["id"] == "milk":
			assert(int(panel.snapshot["chain_batch"]["stock"]["skimmed_milk_g"]) == 72000 * batch)
		if profile["id"] == "oil":
			var packed: Dictionary = panel.snapshot["chain_batch"]["stock"]
			assert(
				(
					int(packed["lubricant_base_g"]) == 0
					and int(packed["plastic_bottle_count"]) == 0
					and int(packed["bottle_cap_count"]) == 0
				)
			)
		await panel.sell_bread()
		assert(panel.sale_state["status"] == "accepted")
		run_tick(tick + 1)
		run_tick(tick + 1)
		await panel.refresh()
		total_revenue += int(profile["quantity"]) * int(profile["price"])
		assert(int(panel.snapshot["cash_kopecks"]) == 200000 + total_revenue - total_cost)
		assert(int(panel.snapshot["cost_kopecks"]) == total_cost)
		assert(int(panel.snapshot["profit_kopecks"]) == total_revenue - total_cost)
		assert(int(panel.snapshot["chain_batch"]["stock"][profile["output"]]) == 0)
		assert(panel.sale_state["status"] == "applied" and panel.sell_button.disabled)
		assert(int(panel.snapshot["completed_batches"]) == batch)
		assert(not panel.buy_button.disabled)
		first_tick = tick + 2
	if profile["id"] == "wood":
		await run_advance_supply(panel, first_tick)
	print("CHAIN SMOKE OK: " + profile["id"])
	quit()


func run_advance_supply(panel: PilotPanel, first_tick: int) -> void:
	var before := int(panel.snapshot["cash_kopecks"])
	var units := panel.supplies.filter(
		func(offer: Dictionary) -> bool: return offer["chain"] == "wood"
	)
	await panel.order_supply(units[1])
	run_tick(first_tick)
	await panel.refresh()
	units = panel.supplies.filter(func(offer: Dictionary) -> bool: return offer["chain"] == "wood")
	assert(
		(
			units[0]["available"]
			and str(panel.supply_buttons.get_child(0).text).begins_with("Заказать про запас")
		)
	)
	await panel.order_supply(units[0])
	run_tick(first_tick + 1)
	run_tick(first_tick + 1)
	await panel.refresh()
	assert(int(panel.snapshot["cash_kopecks"]) == before - 37000)
	assert(
		(
			panel.snapshot["chain_batch"].has("next_order")
			and panel.snapshot["chain_batch"]["stage"] == 0
		)
	)
	for tick in range(first_tick + 2, first_tick + 5):
		run_tick(tick)
	await panel.refresh()
	await panel.sell_bread()
	run_tick(first_tick + 5)
	await panel.refresh()
	assert(
		(
			not panel.snapshot["chain_batch"].has("next_order")
			and panel.snapshot["chain_batch"]["stage"] == 1
		)
	)
	run_tick(first_tick + 6)
	await panel.refresh()
	await panel.sell_bread()
	run_tick(first_tick + 7)
	await panel.refresh()
	assert(int(panel.snapshot["cash_kopecks"]) == before + 19000)
	assert(int(panel.snapshot["completed_batches"]) == 5)
	print("ADVANCE SUPPLY SMOKE OK: reserve, repeat, sell and resume")


func run_mill_trade(panel: PilotPanel, mill_token: String) -> void:
	var mill := PilotClient.new()
	root.add_child(mill)
	assert(mill.configure(panel.api.base_url, panel.api.world, mill_token))
	var mill_before := await mill.fetch("/company")
	assert(mill_before["status"] == 200 and mill_before["data"]["mill_flour_g"] == "7400")
	var offer := await panel.api.fetch("/mill-offer")
	assert(offer["status"] == 200 and offer["data"]["available"])
	assert(offer["data"]["quantity_g"] == "7400" and offer["data"]["price_kopecks"] == "42000")
	assert(offer["data"]["ingredients_kopecks"] == "20400")
	assert(panel.mill_buy_button.visible and not panel.mill_buy_button.disabled)
	assert(panel.remote_buy_button.visible and not panel.remote_buy_button.disabled)
	assert(
		(
			panel.remote_offer["price_kopecks"] == "36000"
			and int(panel.remote_offer["delivery_ticks"]) == 2
		)
	)
	var purchase_id := panel.mill_order_id()
	await panel.order_mill_flour()
	assert(panel.mill_order_state["status"] == "accepted")
	var purchase := {
		"offer_id": "pilot-mill-flour-v1", "quantity": "7400", "max_price_kopecks": "42000"
	}
	var duplicate := await panel.api.submit(purchase_id, "buy_mill_flour", purchase)
	assert(duplicate["status"] == 202)
	await panel.configure_line()
	assert(panel.line_state["status"] == "accepted")

	run_tick(1)
	run_tick(1)
	await panel.refresh()
	assert(
		panel.mill_order_state["status"] == "applied" and panel.line_state["status"] == "applied"
	)
	assert(panel.snapshot["flour_g"] == "7400" and panel.snapshot["cash_kopecks"] == "137600")
	assert(
		panel.snapshot["paper_pack_count"] == "24" and panel.snapshot["packed_bread_count"] == "0"
	)
	assert(panel.sell_button.disabled)
	var mill_after := await mill.fetch("/company")
	assert(mill_after["status"] == 200 and mill_after["data"]["mill_flour_g"] == "0")
	assert(
		int(mill_after["data"]["cash_kopecks"]) == int(mill_before["data"]["cash_kopecks"]) + 42000
	)
	assert(panel.mill_buy_button.disabled)
	var sold_offer := await panel.api.fetch("/mill-offer")
	assert(sold_offer["status"] == 200 and not sold_offer["data"]["available"])
	duplicate = await panel.api.submit(purchase_id, "buy_mill_flour", purchase)
	assert(duplicate["status"] == 202)

	run_tick(2)
	await panel.refresh()
	assert(panel.snapshot["flour_g"] == "200" and panel.snapshot["packed_bread_count"] == "24")
	assert(panel.snapshot["cash_kopecks"] == "123600" and not panel.sell_button.disabled)
	await panel.sell_bread()
	assert(panel.sale_state["status"] == "accepted")
	run_tick(3)
	run_tick(3)
	await panel.refresh()
	assert(panel.sale_state["status"] == "applied" and panel.snapshot["packed_bread_count"] == "0")
	assert(panel.snapshot["cash_kopecks"] == "231600")
	assert(panel.snapshot["completed_batches"] == "1")
	mill.remember_session()
	var seller_panel := PilotScene.instantiate() as PilotPanel
	root.add_child(seller_panel)
	while seller_panel.busy:
		await process_frame
	assert(seller_panel.mill_grain_button.visible and not seller_panel.mill_grain_button.disabled)
	await seller_panel.order_mill_grain()
	assert(seller_panel.mill_grain_state["status"] == "accepted")
	run_tick(4)
	run_tick(4)
	await seller_panel.refresh()
	assert(
		(
			seller_panel.snapshot["mill_grain_g"] == "10000"
			and seller_panel.snapshot["cash_kopecks"] == "212000"
		)
	)
	run_tick(5)
	await seller_panel.refresh()
	await panel.refresh()
	assert(
		(
			seller_panel.snapshot["mill_grain_g"] == "0"
			and seller_panel.snapshot["mill_flour_g"] == "7400"
		)
	)
	assert(panel.mill_buy_button.disabled)
	assert(not seller_panel.mill_list_button.disabled)
	await seller_panel.list_mill_flour()
	run_tick(6)
	await panel.refresh()
	assert(not panel.mill_buy_button.disabled)
	var next_offer := str(panel.mill_offer["offer_id"])
	assert(next_offer != purchase["offer_id"])
	var competitor := PilotClient.new()
	root.add_child(competitor)
	assert(
		competitor.configure(
			panel.api.base_url, panel.api.world, OS.get_environment("EBASH_PILOT_COMPETITOR_TOKEN")
		)
	)
	var second_purchase := purchase.duplicate()
	second_purchase["offer_id"] = next_offer
	var contender := await competitor.submit(
		"competing-purchase", "buy_mill_flour", second_purchase
	)
	assert(contender["status"] == 202)
	await panel.order_mill_flour()
	run_tick(7)
	run_tick(7)
	await panel.refresh()
	assert(panel.mill_order_state["status"] == "applied" and panel.snapshot["flour_g"] == "7600")
	assert(panel.snapshot["packed_bread_count"] == "0")
	var rejection := await competitor.fetch("/commands/competing-purchase")
	assert(
		(
			rejection["data"]["status"] == "rejected"
			and rejection["data"]["reason"] == "MILL_OFFER_UNAVAILABLE"
		)
	)
	var other := await competitor.fetch("/company")
	assert(other["data"]["cash_kopecks"] == "200000" and other["data"]["flour_g"] == "0")
	var fallback := await competitor.submit(
		"fallback-purchase",
		"order_raw",
		{"offer_id": "pilot-inputs-v2", "quantity": "1", "max_price_kopecks": "50400"}
	)
	assert(fallback["status"] == 202)
	assert(not panel.remote_buy_button.disabled)
	var reserve_id := panel.remote_order_id()
	await panel.order_remote_flour()
	assert(panel.remote_order_state["status"] == "accepted")
	var reserve := {
		"offer_id": panel.remote_offer["offer_id"], "quantity": "7400", "max_price_kopecks": "36000"
	}
	var replay := await panel.api.submit(reserve_id, "buy_mill_flour", reserve)
	assert(replay["status"] == 202)
	run_tick(8)
	run_tick(8)
	await panel.refresh()
	assert(panel.snapshot["packed_bread_count"] == "24" and panel.snapshot["flour_g"] == "400")
	assert(panel.snapshot["next_mill_order"]["delivery_tick"] == "10")
	assert(panel.snapshot["next_mill_order"]["price_kopecks"] == "36000")
	await panel.sell_bread()
	run_tick(9)
	await panel.refresh()
	assert(
		panel.snapshot["cash_kopecks"] == "206800" and panel.snapshot["completed_batches"] == "2"
	)
	var mill_final := await mill.fetch("/company")
	assert(
		mill_final["data"]["cash_kopecks"] == "254000" and mill_final["data"]["mill_flour_g"] == "0"
	)
	var other_order := await competitor.fetch("/commands/fallback-purchase")
	assert(other_order["data"]["status"] == "applied")
	assert(
		not panel.snapshot.has("next_mill_order") and panel.snapshot["mill_delivery_tick"] == "10"
	)
	assert(panel.snapshot["flour_g"] == "400" and panel.sell_button.disabled)
	run_tick(10)
	run_tick(10)
	await panel.refresh()
	assert(panel.snapshot["flour_g"] == "7800" and panel.sell_button.disabled)
	run_tick(11)
	await panel.refresh()
	assert(panel.snapshot["packed_bread_count"] == "24" and panel.snapshot["flour_g"] == "600")
	await panel.sell_bread()
	run_tick(12)
	await panel.refresh()
	assert(
		panel.snapshot["cash_kopecks"] == "300800" and panel.snapshot["completed_batches"] == "3"
	)
	assert(
		int(panel.snapshot["profit_kopecks"]) == 100800 + int(panel.snapshot["flour_value_kopecks"])
	)
	print("MILL TRADE SMOKE OK: two deliveries, replenishment and competing purchases")
	quit()
