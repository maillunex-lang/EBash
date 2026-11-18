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
 print("REGION OK")
 quit()
