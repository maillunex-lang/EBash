extends Node

const PATH := "user://preferences.cfg"
var frame_limit := 30
var volume := 100
var pan_speed := 100

func _ready() -> void:
 var config := ConfigFile.new()
 if config.load(PATH) == OK:
  frame_limit = 60 if config.get_value("display","fps",30) == 60 else 30
  volume = read_percent(config,"audio","volume",0,100)
  pan_speed = read_percent(config,"map","pan",50,200)
 apply()

func read_percent(config: ConfigFile, section: String, key: String, low: int, high: int) -> int:
 var value: Variant = config.get_value(section,key,100)
 return clampi(value,low,high) if value is int else 100

func apply() -> void:
 Engine.max_fps = frame_limit
 AudioServer.set_bus_mute(0,volume == 0)
 AudioServer.set_bus_volume_db(0,linear_to_db(maxf(0.001,volume/100.0)))

func save_values(frames: int, sound: int, pan: int) -> Error:
 var config := ConfigFile.new()
 config.set_value("display","fps",frames)
 config.set_value("audio","volume",sound)
 config.set_value("map","pan",pan)
 var result := config.save(PATH)
 if result == OK:
  frame_limit = frames
  volume = sound
  pan_speed = pan
  apply()
 return result
