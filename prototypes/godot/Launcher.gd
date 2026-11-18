extends Control

const PilotScene := preload("res://Pilot.tscn")
const MapScene := preload("res://Main.tscn")

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
 for entry in [["Пекарня · серверная экономика", PilotScene], ["Карта · визуальный прототип", MapScene]]:
  var button := Button.new()
  button.text = entry[0]
  button.custom_minimum_size.y = 68
  button.add_theme_font_size_override("font_size", 22)
  button.pressed.connect(func() -> void: get_tree().change_scene_to_packed(entry[1]))
  column.add_child(button)
 var note := Label.new()
 note.text = "Для пекарни нужен личный доступ к тестовому HTTPS-серверу."
 note.horizontal_alignment = HORIZONTAL_ALIGNMENT_CENTER
 note.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
 column.add_child(note)
