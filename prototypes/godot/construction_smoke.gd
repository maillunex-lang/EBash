extends SceneTree

func _initialize() -> void:
 call_deferred("run")

func run() -> void:
 change_scene_to_file("res://Launcher.tscn")
 await scene_changed
 current_scene.play_button.pressed.emit()
 await scene_changed
 current_scene.city_buttons["ufa"].pressed.emit()
 await scene_changed
 var city := current_scene
 while city.construction.is_empty():
  await process_frame
 city.open_search()
 city.search_query.text_changed.emit("свободная")
 assert(city.search_results.get_child_count() == 3)
 city.search_query.text_changed.emit("6")
 city.search_results.get_child(0).pressed.emit()
 assert(city.bakery_button.visible and city.bakery_button.text == "Построить пекарню")
 await city.construction_action()
 assert(city.construction.site == "6" and city.balance == 170000 and not city.construction.open)
 city.open_hub("company")
 await scene_changed
 var hub := current_scene
 while hub.company_data.is_empty():
  await process_frame
 assert(hub.enterprise_buttons.has("works"))
 var works_label := ""
 for child in hub.body.get_children():
  if child is Label and str(child.text).contains("Приёмка на тике"):
   works_label = child.text
 assert(works_label.contains("участок 6"))
 hub.enterprise_buttons["works"].pressed.emit()
 await scene_changed
 city = current_scene
 while city.construction.is_empty():
  await process_frame
 assert(city.active_lot == 6 and city.district.lots[6].kind == "works")
 await city.construction_action()
 assert(not city.construction.open)
 await city.construction_action()
 assert(city.construction.open and city.district.lots[6].kind == "bakery")
 city.open_hub("company")
 await scene_changed
 hub = current_scene
 while hub.company_data.is_empty():
  await process_frame
 assert(hub.enterprise_buttons.has("manage"))
 hub.enterprise_buttons["manage"].pressed.emit()
 await scene_changed
 var panel := current_scene
 while panel.busy:
  await process_frame
 await panel.order_mill_flour()
 await panel.advance_demo_tick()
 await panel.advance_demo_tick()
 assert(int(panel.profile_stock().get("bread_loaf",0)) == 24)
 print("CONSTRUCTION OK")
 quit()
