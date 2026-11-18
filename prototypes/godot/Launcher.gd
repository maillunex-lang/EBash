extends Control

const RegionScene := preload("res://Region.tscn")

var play_button: Button
var settings_button := Button.new()
var settings := VBoxContainer.new()
var frames := OptionButton.new()
var volume := HSlider.new()
var pan := HSlider.new()
var notice := Label.new()
var save_button := Button.new()

func _ready() -> void:
 var background := ColorRect.new()
 background.color = Color("101923")
 background.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
 add_child(background)
 var center := CenterContainer.new()
 center.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
 add_child(center)
 var column := VBoxContainer.new()
 column.custom_minimum_size.x = 470
 column.add_theme_constant_override("separation", 24)
 center.add_child(column)
 var heading := Label.new()
 heading.text = "EBash · тестовая версия"
 heading.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
 heading.add_theme_font_size_override("font_size", 32)
 column.add_child(heading)
 play_button = Button.new()
 play_button.text = "Играть"
 play_button.custom_minimum_size.y = 68
 play_button.add_theme_font_size_override("font_size", 22)
 play_button.pressed.connect(func() -> void:
  get_tree().set_meta("ebash_hub_page", "region")
  get_tree().change_scene_to_packed(RegionScene)
 )
 column.add_child(play_button)
 settings_button.text = "Настройки"
 settings_button.custom_minimum_size.y = 56
 column.add_child(settings_button)
 column.add_child(settings)
 settings.hide()
 settings.add_theme_constant_override("separation",10)
 frames.add_item("30 кадров/с",30)
 frames.add_item("60 кадров/с",60)
 settings.add_child(frames)
 add_percent("Громкость",volume,0,100)
 add_percent("Скорость перемещения карты",pan,50,200)
 var actions := HBoxContainer.new()
 settings.add_child(actions)
 save_button.text = "Сохранить"
 save_button.custom_minimum_size = Vector2(200,52)
 actions.add_child(save_button)
 var cancel := Button.new()
 cancel.text = "Назад"
 cancel.custom_minimum_size = Vector2(160,52)
 actions.add_child(cancel)
 settings.add_child(notice)
 notice.hide()
 settings_button.pressed.connect(func(): show_settings(true))
 cancel.pressed.connect(func(): show_settings(false))
 save_button.pressed.connect(save_settings)

func add_percent(caption: String, slider: HSlider, minimum: int, maximum: int) -> void:
 var label := Label.new()
 label.add_theme_font_size_override("font_size",20)
 settings.add_child(label)
 slider.min_value = minimum
 slider.max_value = maximum
 slider.step = 5
 slider.custom_minimum_size.y = 46
 settings.add_child(slider)
 slider.value_changed.connect(func(value: float): label.text = "%s · %d%%" % [caption,int(value)])
 slider.value_changed.emit(slider.value)

func show_settings(open: bool) -> void:
 play_button.visible = not open
 settings_button.visible = not open
 settings.visible = open
 notice.hide()
 if open:
  var prefs := get_node("/root/Preferences")
  frames.select(frames.get_item_index(prefs.frame_limit))
  volume.value = prefs.volume
  pan.value = prefs.pan_speed

func save_settings() -> void:
 var prefs := get_node("/root/Preferences")
 if prefs.save_values(frames.get_selected_id(),int(volume.value),int(pan.value)) == OK:
  show_settings(false)
 else:
  notice.text = "Не удалось сохранить настройки устройства."
  notice.show()
