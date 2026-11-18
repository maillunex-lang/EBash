extends SceneTree

func _initialize() -> void:
 call_deferred("run")

func run() -> void:
 change_scene_to_file("res://Launcher.tscn")
 await scene_changed
 var menu := current_scene
 menu.settings_button.pressed.emit()
 if "--verify" in OS.get_cmdline_user_args():
  assert(menu.frames.get_selected_id() == 60 and menu.volume.value == 0 and menu.pan.value == 150)
  assert(Engine.max_fps == 60 and AudioServer.is_bus_mute(0))
  menu.show_settings(false)
  menu.play_button.pressed.emit()
  await scene_changed
  for id in ["ufa","sterlitamak","salavat"]:
   current_scene.city_buttons[id].pressed.emit()
   await scene_changed
   var city := current_scene
   var start: Vector3 = city.camera.position
   city._pan_camera(Vector2(10,0))
   assert(city.camera.position.is_equal_approx(start + Vector3(-10,0,10)*city.camera.size/900.0*1.5))
   assert(Engine.max_fps == 60)
   city.open_hub("region")
   await scene_changed
 else:
  menu.frames.select(1)
  menu.volume.value = 0
  menu.pan.value = 150
  menu.save_button.pressed.emit()
  assert(not menu.settings.visible)
 print("PREFERENCES OK")
 quit()
