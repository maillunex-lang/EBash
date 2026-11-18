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
 var scene = load("res://Main.tscn").instantiate()
 root.add_child.call_deferred(scene)
 await process_frame
 await process_frame
 assert(scene.buildings.size() == 49)
 scene.show_factory()
 assert(scene.factory_view == "production" and scene.slot_buttons.size() == 9)
 scene.select_cell(4)
 assert(scene.selected == 4 and scene.slot_buttons[4].disabled)
 scene.set_factory_view("chain")
 assert(scene.factory_view == "chain" and scene.slot_buttons.size() == 9)
 scene.select_cell(1)
 assert(scene.selected == 1 and scene.factory_view == "chain")
 scene.set_factory_view("production")
 var balance = scene.cash
 scene.advance_tick()
 assert(scene.tick == 1 and scene.cash == balance+1250)
 scene.toggle_mode()
 assert(scene.mode)
 scene.show_republic()
 assert(not scene.city.visible)
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
 var before_zoom: float = scene.camera.size
 scene._unhandled_input(pinch)
 assert(scene.camera.size < before_zoom)
 touch_b.pressed = false
 scene._unhandled_input(touch_b)
 touch_a.pressed = false
 scene._unhandled_input(touch_a)
 assert(scene.touches.is_empty())
 scene._layout_panel()
 assert(scene.panel.offset_right - scene.panel.offset_left <= 800.0)
 print("SMOKE OK: city, factory, selection, demo tick, company, republic")
 quit()
