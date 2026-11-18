extends SceneTree
const District := preload("res://District.gd")

func check_districts() -> void:
 for district_seed in range(128):
  var map := District.generate(district_seed)
  assert(map == District.generate(district_seed))
  assert(map != District.generate(district_seed+1))
  var plots: Array[Rect2] = []
  var sites: Dictionary = {}
  assert(map.lots.size() == 25)
  for lot: Dictionary in map.lots:
   var plot: Rect2 = lot.plot
   for other in plots:
    assert(not plot.intersects(other))
   plots.append(plot)
   assert(plot.size.x > 6.4 and plot.size.y > 5.4)
   for x: int in map.xs:
    assert(plot.end.x <= x-1.25 or plot.position.x >= x+1.25)
   for z: int in map.zs:
    assert(plot.end.y <= z-1.25 or plot.position.y >= z+1.25)
   assert(int(plot.position.y-1.25) in map.zs)
   if lot.kind in ["bakery","near","remote"]:
    assert(not sites.has(lot.kind))
    sites[lot.kind] = plot.get_center()
  assert(sites.size() == 3)
  var bakery: Vector2 = sites.bakery
  assert(bakery.distance_to(sites.near) < bakery.distance_to(sites.remote))

func _initialize() -> void:
 check_districts()
 set_meta("ebash_city_route",true)
 set_meta("ebash_city_id","ufa")
 var scene = load("res://Main.tscn").instantiate()
 root.add_child.call_deferred(scene)
 await process_frame
 await process_frame
 assert(scene.city_id == "ufa" and scene.district.lots.size() == 25)
 assert(scene.city_card.visible)
 scene.open_search()
 assert(scene.search_results.get_child_count() == 25)
 scene.search_query.text_changed.emit("12")
 assert(scene.search_results.get_child_count() == 1)
 scene.search_results.get_child(0).pressed.emit()
 assert(scene.active_lot == 12)
 var before_zoom: float = scene.camera.size
 var touch_a := InputEventScreenTouch.new()
 touch_a.index = 0
 touch_a.pressed = true
 touch_a.position = Vector2(200, 200)
 scene._unhandled_input(touch_a)
 var touch_b := InputEventScreenTouch.new()
 touch_b.index = 1
 touch_b.pressed = true
 touch_b.position = Vector2(300, 200)
 scene._unhandled_input(touch_b)
 var pinch := InputEventScreenDrag.new()
 pinch.index = 1
 pinch.position = Vector2(350, 200)
 scene._unhandled_input(pinch)
 assert(scene.camera.size < before_zoom)
 touch_b.pressed = false
 scene._unhandled_input(touch_b)
 touch_a.pressed = false
 scene._unhandled_input(touch_a)
 assert(scene.touches.is_empty())
 scene.open_hub("company")
 await scene_changed
 assert(current_scene.company_tab.disabled)
 print("SMOKE OK: districts, city map, search, camera, company hub")
 quit()
