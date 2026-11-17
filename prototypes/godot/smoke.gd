extends SceneTree
func _initialize() -> void:
 var scene = load("res://Main.tscn").instantiate()
 root.add_child.call_deferred(scene)
 await process_frame
 await process_frame
 assert(scene.buildings.size() == 49)
 scene.show_factory()
 scene.select_cell(4)
 assert(scene.selected == 4)
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
