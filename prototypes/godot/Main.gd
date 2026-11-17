extends Node3D
var camera: Camera3D
var city: Node3D
var canvas: Control
var panel: PanelContainer
var status: Label
var fps: Label
var mode := false
var dragging := false
var last := Vector2.ZERO
var moved := false
var buildings: Array[Node3D] = []
var selected := 0
var cash := 250000
var tick := 0
var linked := -1
var roles := ["Молоко", "Бутылки", "Крышки", "Розлив", "Розлив", "Коктейль", "Продажа", "Продажа", "Продажа"]

func box(parent: Node3D, pos: Vector3, size: Vector3, color: Color) -> MeshInstance3D:
 var m := MeshInstance3D.new()
 var mesh := BoxMesh.new()
 mesh.size = size
 m.mesh = mesh
 m.position = pos
 var mat := StandardMaterial3D.new()
 mat.albedo_color = color
 mat.roughness = 0.85
 m.material_override = mat
 parent.add_child(m)
 return m

func _ready() -> void:
 Engine.max_fps = 30
 city = Node3D.new()
 add_child(city)
 box(city, Vector3(0,-0.2,0), Vector3(90,0.3,90), Color("8caa84"))
 for i in range(-3,4):
  box(city,Vector3(i*12,0,0),Vector3(2.4,0.12,86),Color("485664"))
  box(city,Vector3(0,0.01,i*12),Vector3(86,0.12,2.4),Color("485664"))
  for j in range(-3,4):
   var b := Node3D.new()
   b.position = Vector3(i*12+5,0,j*12+5)
   city.add_child(b)
   var n := (i+3)*7+j+3
   var h := 2.5+float(n%5)*1.2
   var c := Color("d8d4bf") if n%3 else Color("69abb4")
   box(b,Vector3(0,h/2,0),Vector3(6,h,5),c)
   box(b,Vector3(0,h+0.2,0),Vector3(6.3,0.4,5.3),Color("344656"))
   for floor_id in range(int(h/1.3)):
    for w in range(3):
     box(b,Vector3(-2+w*2,0.8+floor_id*1.3,2.52),Vector3(0.8,0.7,0.06),Color("c6e7ed"))
   buildings.append(b)
 var light := DirectionalLight3D.new()
 light.rotation_degrees = Vector3(-55,-30,0)
 light.light_energy = 1.25
 light.shadow_enabled = true
 add_child(light)
 var env := WorldEnvironment.new()
 env.environment = Environment.new()
 env.environment.background_mode = Environment.BG_COLOR
 env.environment.background_color = Color("b7d1df")
 env.environment.ambient_light_source = Environment.AMBIENT_SOURCE_COLOR
 env.environment.ambient_light_color = Color("d6e4ed")
 env.environment.ambient_light_energy = 0.65
 add_child(env)
 camera = Camera3D.new()
 camera.projection = Camera3D.PROJECTION_ORTHOGONAL
 camera.size = 62
 camera.position = Vector3(55,58,55)
 add_child(camera)
 camera.look_at(Vector3.ZERO)
 make_ui()

func button(text: String, callback: Callable, parent: Node) -> Button:
 var b := Button.new()
 b.text = text
 b.custom_minimum_size = Vector2(140,48)
 b.pressed.connect(callback)
 parent.add_child(b)
 return b

func label(text: String, parent: Node, size: int = 20) -> Label:
 var l := Label.new()
 l.text = text
 l.add_theme_font_size_override("font_size",size)
 parent.add_child(l)
 return l

func make_ui() -> void:
 var layer := CanvasLayer.new()
 add_child(layer)
 canvas = Control.new()
 canvas.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
 canvas.mouse_filter = Control.MOUSE_FILTER_IGNORE
 layer.add_child(canvas)
 var top := PanelContainer.new()
 top.set_anchors_and_offsets_preset(Control.PRESET_TOP_WIDE)
 top.offset_bottom = 64
 canvas.add_child(top)
 var row := HBoxContainer.new()
 top.add_child(row)
 label("  EBASH / УФА    •    GODOT",row,24)
 status = label("",row)
 status.size_flags_horizontal = Control.SIZE_EXPAND_FILL
 button("Карта / Компания",toggle_mode,row)
 button("Тик +1",advance_tick,row)
 fps = label("",row)
 var nav := VBoxContainer.new()
 nav.position = Vector2(16,86)
 canvas.add_child(nav)
 button("Республика",show_republic,nav)
 button("Уфа",func(): city.visible=true; panel.hide(),nav)
 button("Мой завод",func(): show_factory(),nav)
 button("+ Масштаб",func(): camera.size=maxf(22,camera.size-6),nav)
 button("− Масштаб",func(): camera.size=minf(110,camera.size+6),nav)
 button("30 / 60 FPS",func(): Engine.max_fps=60 if Engine.max_fps==30 else 30,nav)
 var note := PanelContainer.new()
 note.set_anchors_and_offsets_preset(Control.PRESET_BOTTOM_WIDE)
 note.offset_top = -54
 canvas.add_child(note)
 label("  ПРОТОТИП • Локальные демоданные • Перетаскивайте карту / щипок для масштаба",note,18)
 panel = PanelContainer.new()
 panel.set_anchors_and_offsets_preset(Control.PRESET_CENTER)
 panel.offset_left=-360
 panel.offset_right=440
 panel.offset_top=-240
 panel.offset_bottom=270
 canvas.add_child(panel)
 panel.hide()
 advance_status()

func clear_panel() -> VBoxContainer:
 for child in panel.get_children():
  panel.remove_child(child)
  child.queue_free()
 var v := VBoxContainer.new()
 panel.add_child(v)
 panel.show()
 return v

func show_republic() -> void:
 city.visible = false
 var v := clear_panel()
 label("РЕСПУБЛИКА БАШКОРТОСТАН",v,28)
 label("Навигация по городам • условная обзорная схема",v)
 for name in ["Уфа", "Стерлитамак", "Салават", "Нефтекамск"]:
  button(name,func(): city.visible=true; panel.hide(); status.text="  "+name+" • демонстрационная сцена",v)
 label("В прототипе города используют одну тестовую сцену.",v,16)
 button("Закрыть",func(): panel.hide();city.visible=true,v)

func toggle_mode() -> void:
 mode = not mode
 if not mode:
  panel.hide()
  city.visible=true
  return
 var v := clear_panel()
 label("КОМПАНИЯ / ОБЗОР",v,28)
 label("Баланс: %s ₽\nПредприятий: 1\nМолочный завод • загрузка 72%%\nПоследний демотик: %s" % [cash,tick],v)
 button("Управлять заводом",show_factory,v)
 button("Назад к карте",toggle_mode,v)

func show_factory() -> void:
 var v := clear_panel()
 label("МОЛОЧНЫЙ ЗАВОД / ПРОИЗВОДСТВО",v,26)
 label("Обзор  •  Производство  •  Персонал  •  Склады  •  Финансы",v,17)
 var row := HBoxContainer.new()
 v.add_child(row)
 var grid := GridContainer.new()
 grid.columns=3
 row.add_child(grid)
 for i in range(9):
  var index := i
  button("%s\n%s" % [i+1,roles[i]],func(): select_cell(index),grid)
 var details := VBoxContainer.new()
 row.add_child(details)
 label("  Выбрана ячейка %s\n  %s\n  Качество: 38 / 100\n  Загрузка: 72%%" % [selected+1,roles[selected]],details)
 button("Сменить продукт",func(): roles[selected]="Коктейль" if roles[selected]!="Коктейль" else "Розлив";show_factory(),details)
 button("Начать связь",func(): linked=selected;status.text="  Выберите ячейку-получатель",details)
 button("Модернизация −5000 ₽",func():
  if cash>=5000:
   cash-=5000
   advance_status(),details)
 label("Деморецепты и показатели условны. Экономика сервера не реализована.",v,15)
 button("Закрыть",func(): panel.hide(),v)

func select_cell(index: int) -> void:
 if linked>=0:
  status.text="  Связь: %s → %s (демо)" % [linked+1,index+1]
  linked=-1
 selected=index
 show_factory()

func advance_status() -> void:
 status.text="  %s ₽   •   Тик %s   •   ДЕМО" % [cash,tick]

func advance_tick() -> void:
 tick+=1
 cash+=1250
 advance_status()

func _process(_delta: float) -> void:
 fps.text=" %s FPS " % Engine.get_frames_per_second()

func _unhandled_input(event: InputEvent) -> void:
 if event is InputEventMagnifyGesture:
  camera.size=clampf(camera.size/event.factor,22,110)
 elif event is InputEventMouseButton:
  if event.button_index==MOUSE_BUTTON_WHEEL_UP:
   camera.size=maxf(22,camera.size-3)
  elif event.button_index==MOUSE_BUTTON_WHEEL_DOWN:
   camera.size=minf(110,camera.size+3)
  elif event.button_index==MOUSE_BUTTON_LEFT:
   dragging=event.pressed
   if event.pressed:
    last=event.position
    moved=false
   elif not moved and city.visible:
    show_factory()
 elif event is InputEventMouseMotion and dragging:
  var d: Vector2=event.position-last
  last=event.position
  moved=moved or d.length()>3
  camera.position += Vector3(-d.x-d.y,0,d.x-d.y)*camera.size/900.0
