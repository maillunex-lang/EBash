extends SceneTree

func _initialize() -> void:
 call_deferred("run")

func run() -> void:
 change_scene_to_file("res://Launcher.tscn")
 await scene_changed
 current_scene.play_button.pressed.emit()
 await scene_changed
 for id in ["sterlitamak","salavat","ufa"]:
  current_scene.city_buttons[id].pressed.emit()
  await scene_changed
  assert(current_scene.city_id == id and current_scene.district.lots.size() == 25)
  if id != "ufa":
   for lot: Dictionary in current_scene.district.lots:
    assert(lot.kind in ["homes","park"])
  var city := current_scene
  city.open_search()
  assert(city.search_results.get_child_count() == 25)
  city.search_query.text_changed.emit("  12 ")
  assert(city.search_results.get_child_count() == 1)
  city.search_results.get_child(0).pressed.emit()
  assert(city.active_lot == 12 and city.city_card.visible)
  var target: Vector3 = city.lot_bounds[12].get_center()
  target.y = 0
  var screen_center: Vector2 = city.get_viewport().get_visible_rect().get_center()
  assert(city.camera.unproject_position(target).distance_to(screen_center) < 1)
  city.open_search()
  city.search_query.text_changed.emit("МЕЛЬНИЦА")
  if id == "ufa":
   assert(city.search_results.get_child_count() == 2)
  else:
   assert(city.search_results.get_child(0).text == "Совпадений нет")
  city.close_search()
  current_scene.camera.size = 41
  current_scene.open_hub("company")
  await scene_changed
  assert(current_scene.company_tab.disabled)
  current_scene.region_tab.pressed.emit()
  current_scene.city_buttons[id].pressed.emit()
  await scene_changed
  assert(current_scene.camera.size == 41)
  current_scene.open_hub("region")
  await scene_changed
 await check_company_enterprises()
 print("REGION OK")
 quit()

func check_company_enterprises() -> void:
 var hub := current_scene
 hub.page("Моя компания",true)
 hub.summary = Label.new()
 hub.summary.add_theme_font_size_override("font_size",22)
 hub.body.add_child(hub.summary)
 hub.fill_company({
  "cash_kopecks":"200000",
  "last_committed_tick":"0",
  "completed_batches":"0",
  "game_production":{"construction":{"sites":{"6":30000,"12":40000,"18":35000},"duration":2,"site":"","ready_tick":0,"open":false}}
 })
 assert(hub.enterprise_buttons.has("6") and hub.enterprise_buttons.has("12") and hub.enterprise_buttons.has("18"))
 assert(hub.summary.text.contains("2000.00"))
 hub.enterprise_buttons["6"].pressed.emit()
 await scene_changed
 var city := current_scene
 assert(city.city_id == "ufa" and city.active_lot == 6)
 city.open_hub("company")
 await scene_changed
 hub = current_scene
 hub.page("Моя компания",true)
 hub.summary = Label.new()
 hub.summary.add_theme_font_size_override("font_size",22)
 hub.body.add_child(hub.summary)
 hub.fill_company({
  "cash_kopecks":"170000",
  "last_committed_tick":"1",
  "completed_batches":"0",
  "game_production":{"construction":{"sites":{"6":30000,"12":40000,"18":35000},"duration":2,"site":"6","ready_tick":3,"open":false}}
 })
 assert(hub.enterprise_buttons.has("works"))
 var works_text := ""
 for child in hub.body.get_children():
  if child is Label and str(child.text).contains("Приёмка на тике 3"):
   works_text = child.text
 assert(works_text.contains("осталось 2 тика"))
 hub.enterprise_buttons["works"].pressed.emit()
 await scene_changed
 city = current_scene
 assert(city.active_lot == 6)
 city.open_hub("company")
 await scene_changed
 hub = current_scene
 hub.page("Моя компания",true)
 hub.summary = Label.new()
 hub.summary.add_theme_font_size_override("font_size",22)
 hub.body.add_child(hub.summary)
 var open_data := {
  "cash_kopecks":"170000",
  "last_committed_tick":"3",
  "completed_batches":"0",
  "revenue_kopecks":"0",
  "cost_kopecks":"30000",
  "profit_kopecks":"0",
  "game_production":{"construction":{"sites":{"6":30000,"12":40000,"18":35000},"duration":2,"site":"6","ready_tick":3,"open":true}}
 }
 hub.fill_company(open_data)
 assert(hub.enterprise_buttons.has("manage"))
 hub.show_finance()
 assert(hub.title.text == "Финансы")
 hub.show_market()
 assert(hub.title.text == "Рынок")
 hub.page("Моя компания",true)
 hub.summary = Label.new()
 hub.summary.add_theme_font_size_override("font_size",22)
 hub.body.add_child(hub.summary)
 hub.fill_company(open_data)
 hub.enterprise_buttons["manage"].pressed.emit()
 await scene_changed
 assert(current_scene.has_method("order_mill_flour"))
