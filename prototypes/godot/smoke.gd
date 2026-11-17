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
 print("SMOKE OK: city, factory, selection, demo tick, company, republic")
 quit()
