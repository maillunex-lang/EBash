extends Node3D

const PilotClient := preload("res://PilotClient.gd")

const District := preload("res://District.gd")
const CityCatalog := preload("res://CityCatalog.gd")
var city_id := "ufa"
var district: Dictionary
var lot_bounds: Array[AABB] = []
var construction: Dictionary = {}
var company_tick := 0
var balance := 0
var building_busy := false
var build_notice: Label
var active_lot := -1
var object_name: Label
var selection: MeshInstance3D

var bakery_button: Button
var bakery_status: Label
var city_card: PanelContainer
var search_card: PanelContainer
var search_query: LineEdit
var search_results: VBoxContainer
var city_api: PilotClient
var camera: Camera3D
var city: Node3D
var canvas: Control
var panel: PanelContainer
var status: Label
var fps: Label
var mode := false
var dragging := false
var touches: Dictionary = {}
var pinch_distance := 0.0
var touch_moved := false
var press_position := Vector2.ZERO
var last := Vector2.ZERO
var moved := false
var buildings: Array[Node3D] = []
var selected := 0
var cash := 250000
var tick := 0
var linked := -1
var roles := [
 "Молоко", "Бутылки", "Крышки", "Розлив", "Розлив",
 "Коктейль", "Продажа", "Продажа", "Продажа"
]

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
 city_id = str(get_tree().get_meta("ebash_city_id","ufa"))
 city = Node3D.new()
 add_child(city)
 if get_tree().has_meta("ebash_city_route"):
  build_district()
 else:
  build_demo_city()
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
 if get_tree().has_meta("ebash_city_route"):
  make_game_city()
 else:
  make_ui()

func build_demo_city() -> void:
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

func build_district() -> void:
 district = District.generate(CityCatalog.CITIES[city_id].seed,CityCatalog.CITIES[city_id].bakery_world)
 active_lot = 12
 if not construction.is_empty():
  for id: String in construction.sites:
   var kind := "site"
   if id == construction.site:
    kind = "bakery" if construction.open else "works"
   district.lots[int(id)].kind = kind
 var xs: PackedInt32Array = district.xs
 var zs: PackedInt32Array = district.zs
 box(city, Vector3(0,-0.25,0), Vector3(80,0.4,80), Color("96ad83"))
 for x in xs:
  box(city,Vector3(x,0,(zs[0]+zs[-1])*0.5),Vector3(2.5,0.08,zs[-1]-zs[0]+2.5),Color("485664"))
 for z in zs:
  box(city,Vector3((xs[0]+xs[-1])*0.5,0.01,z),Vector3(xs[-1]-xs[0]+2.5,0.08,2.5),Color("485664"))
 for lot: Dictionary in district.lots:
  var plot: Rect2 = lot.plot
  var center := plot.get_center()
  var kind: String = lot.kind
  var height: float = lot.height
  var base := Vector3(center.x,0,center.y)
  var park := kind == "park"
  var pad := Vector3(plot.size.x-0.5,0.15,plot.size.y-0.5)
  box(city,base,pad,Color("83a574") if park else Color("c3bbab"))
  if kind == "works":
   box(city,base+Vector3.UP*0.45,Vector3(6,0.9,5),Color("ba8558"))
  elif kind == "site":
   box(city,base+Vector3.UP*0.1,Vector3(6,0.15,5),Color("d3ad79"))
  elif park:
   box(city,base+Vector3.UP*1.3,Vector3(3,2.5,3),Color("477151"))
  else:
   box(city,base+Vector3.UP*height/2,Vector3(6,height,5),Color("d5c2a5"))
   box(city,base+Vector3.UP*height,Vector3(6.4,0.38,5.4),Color("526471"))
   var drive := (plot.size.y-5)*0.5
   box(city,base+Vector3(0,0.10,-2.5-drive/2),Vector3(2,0.06,drive),Color("787d7c"))
  if kind in ["near","remote","bakery","site","works"]:
   var sign_node := Label3D.new()
   sign_node.text = District.NAMES[kind]
   sign_node.billboard = BaseMaterial3D.BILLBOARD_ENABLED
   sign_node.no_depth_test = true
   sign_node.pixel_size = 0.03
   sign_node.position = base + Vector3.UP*(height+1.6)
   city.add_child(sign_node)
   if kind == "bakery":
    active_lot = lot_bounds.size()
   elif kind in ["near","remote"]:
    box(city,base+Vector3(1.8,height+0.8,0),Vector3(1.3,1.6,1.3),Color("b9c6c7"))
  lot_bounds.append(AABB(Vector3(plot.position.x,0,plot.position.y),Vector3(plot.size.x,0.16,plot.size.y)))
 selection = box(city,Vector3.ZERO,Vector3.ONE,Color("f0ba56"))

func pick_lot(screen: Vector2) -> void:
 var origin := camera.project_ray_origin(screen)
 var direction := camera.project_ray_normal(screen)
 var nearest := INF
 var chosen := -1
 for i in range(lot_bounds.size()):
  var lot: Dictionary = district.lots[i]
  var center := lot_bounds[i].get_center()
  var park: bool = lot.kind == "park"
  var height := 2.55 if park else float(lot.height) + (1.6 if lot.kind in ["near","remote"] else 0.19)
  var width := 3.0 if park else 6.4
  var depth := 3.0 if park else 5.4
  if lot.kind in ["site","works"]:
   height = 0.9 if lot.kind == "works" else 0.2
  var body := AABB(Vector3(center.x-width/2,0,center.z-depth/2),Vector3(width,height,depth))
  for bounds: AABB in [lot_bounds[i],body]:
   var hit: Variant = bounds.intersects_ray(origin,direction)
   if hit != null and origin.distance_squared_to(hit) < nearest:
    chosen = i
    nearest = origin.distance_squared_to(hit)
 if chosen >= 0:
  select_lot(chosen)

func select_lot(index: int) -> void:
 active_lot = index
 var kind: String = district.lots[index].kind
 var bounds := lot_bounds[index]
 selection.position = bounds.get_center()
 selection.scale = Vector3(bounds.size.x-0.2,0.12,bounds.size.z-0.2)
 selection.position.y = 0.008
 object_name.text = District.NAMES[kind]
 bakery_status.visible = kind == "bakery"
 bakery_button.visible = kind in ["bakery","near","remote"]
 bakery_button.text = "Открыть пекарню" if kind == "bakery" else "Посмотреть предложение"

 if not construction.is_empty() and not construction.open:
  bakery_button.visible = kind == "works" or (kind == "site" and construction.site == "")
  if kind == "site":
   var price := int(construction.sites[str(index)])
   object_name.text = "Участок %s · %s ₽ · %s тика" % [index,price/100,construction.duration]
   bakery_button.text = "Построить пекарню"
   bakery_button.disabled = building_busy or balance < price
  elif kind == "works":
   object_name.text = "Стройка · приёмка на тике %s" % construction.ready_tick
   bakery_button.text = "Следующий тик"
   bakery_button.disabled = building_busy

func construction_action() -> void:
 if building_busy:
  return
 building_busy = true
 bakery_button.disabled = true
 var site := str(active_lot)
 var next_tick := company_tick+1
 var command_id := "build-%s-%s" % [site,next_tick]
 var starting: bool = construction.site == ""
 var error_text := ""
 if starting:
  var result := await city_api.submit(command_id,"build_bakery",{"offer_id":site,"max_price_kopecks":str(int(construction.sites[site]))})
  if result.status == 202:
   next_tick = int(result.data.target_tick)
  else:
   error_text = "Сервер не подтвердил заявку."
 if error_text == "":
  var advanced := await city_api.send(HTTPClient.METHOD_POST,"/demo-tick/%s" % next_tick)
  if advanced.status != 200:
   error_text = "Не удалось подтвердить тик. Повторите действие."
  elif starting:
   var receipt := await city_api.fetch("/commands/"+command_id)
   if receipt.data.get("status","") != "applied":
    error_text = "Стройка не началась. Проверьте участок и баланс."
 building_busy = false
 await refresh_bakery()
 select_lot(active_lot)
 if error_text != "":
  build_notice.text = error_text

func remember_camera() -> void:
 get_tree().set_meta("city_camera_"+city_id,{"position":camera.position,"zoom":camera.size})

func open_hub(tab: String) -> void:
 remember_camera()
 get_tree().set_meta("ebash_hub_page",tab)
 get_tree().change_scene_to_file("res://Region.tscn")

func open_lot() -> void:
 remember_camera()
 var kind: String = district.lots[active_lot].kind
 if kind in ["site","works"]:
  await construction_action()
  return
 if kind in ["near","remote"]:
  get_tree().set_meta("ebash_map_supplier",kind)
 get_tree().change_scene_to_file("res://Pilot.tscn")

func make_city_search(layer: CanvasLayer) -> void:
 search_card = PanelContainer.new()
 search_card.position = city_card.position
 layer.add_child(search_card)
 var column := VBoxContainer.new()
 search_card.add_child(column)
 label("Объекты · %s" % CityCatalog.CITIES[city_id].title,column,24)
 search_query = LineEdit.new()
 search_query.placeholder_text = "Название или номер участка"
 search_query.custom_minimum_size = Vector2(340,48)
 search_query.add_theme_font_size_override("font_size",20)
 column.add_child(search_query)
 var scroll := ScrollContainer.new()
 scroll.custom_minimum_size = Vector2(340,220)
 column.add_child(scroll)
 search_results = VBoxContainer.new()
 search_results.size_flags_horizontal = Control.SIZE_EXPAND_FILL
 scroll.add_child(search_results)
 search_query.text_changed.connect(refresh_search)
 button("Назад к карте",close_search,column)
 search_card.hide()

func open_search() -> void:
 city_card.hide()
 search_card.show()
 search_query.clear()
 refresh_search("")
 search_query.grab_focus()

func close_search() -> void:
 search_query.release_focus()
 search_card.hide()
 city_card.show()

func refresh_search(query: String) -> void:
 for child in search_results.get_children():
  search_results.remove_child(child)
  child.queue_free()
 var term := query.strip_edges().to_lower()
 for i in range(district.lots.size()):
  var name_text: String = District.NAMES[district.lots[i].kind]
  var caption := "%s · участок %d" % [name_text,i]
  if term.is_valid_int() and str(i) != term:
   continue
  if not term.is_empty() and not caption.to_lower().contains(term):
   continue
  button(caption,focus_search_result.bind(i),search_results)
 if search_results.get_child_count() == 0:
  label("Совпадений нет",search_results)

func focus_search_result(index: int) -> void:
 close_search()
 select_lot(index)
 var center := lot_bounds[index].get_center()
 var forward := -camera.global_transform.basis.z
 var ground := camera.position - forward * (camera.position.y / forward.y)
 camera.position += Vector3(center.x-ground.x,0,center.z-ground.z)

func apply_focus_lot() -> void:
 if not get_tree().has_meta("ebash_focus_lot"):
  return
 var index := int(get_tree().get_meta("ebash_focus_lot"))
 get_tree().remove_meta("ebash_focus_lot")
 if index < 0 or index >= lot_bounds.size():
  return
 focus_search_result(index)

func make_game_city() -> void:
 var layer := CanvasLayer.new()
 add_child(layer)
 city_card = PanelContainer.new()
 city_card.position = Vector2(20, 20)
 layer.add_child(city_card)
 var controls := VBoxContainer.new()
 controls.add_theme_constant_override("separation", 12)
 city_card.add_child(controls)
 label(CityCatalog.CITIES[city_id].title, controls, 26)
 object_name = label("", controls, 22)
 bakery_status = label("Пекарня · проверяем состояние…", controls)
 bakery_button = button("Открыть пекарню", open_lot, controls)
 build_notice = label("", controls, 18)
 build_notice.hide()
 select_lot(active_lot)
 button("Найти объект",open_search,controls)
 make_city_search(layer)
 var nav := HBoxContainer.new()
 controls.add_child(nav)
 button("Республика",open_hub.bind("region"),nav)
 button("Компания",open_hub.bind("company"),nav)
 var view: Dictionary = get_tree().get_meta("city_camera_"+city_id,{})
 if not view.is_empty():
  camera.position = view.position
  camera.size = view.zoom
 if not CityCatalog.CITIES[city_id].bakery_world:
  apply_focus_lot()
  return
 city_api = PilotClient.new()
 add_child(city_api)
 await refresh_bakery()
 apply_focus_lot()

func refresh_bakery() -> void:
 if not city_api.configure_from_session():
  bakery_status.text = "Пекарня · подключитесь внутри"
  return
 var response := await city_api.fetch("/company")
 if response["status"] != 200:
  bakery_status.text = "Пекарня · сервер недоступен"
  return
 city_api.remember_session()
 var company: Dictionary = response["data"]
 var production: Dictionary = company.get("game_production", {})
 var state: Dictionary = production.get("construction",{})
 company_tick = int(company.get("last_committed_tick",0))
 balance = int(company.get("cash_kopecks",0))
 if state != construction:
  construction = state
  var index := int(construction.site) if construction.get("site","") != "" else active_lot
  for child in city.get_children():
   city.remove_child(child)
   child.queue_free()
  lot_bounds.clear()
  build_district()
  bakery_button.disabled = false
  select_lot(index)
  if search_card.visible:
   refresh_search(search_query.text)
 if not construction.is_empty():
  build_notice.show()
  build_notice.text = "Баланс %s ₽ · тик %s" % [balance/100,company_tick]
  if construction.site == "":
   build_notice.text += "\nВыберите свободную площадку на карте."
 var stock: Dictionary = production.get("stock", {})
 var delivery: Dictionary = production.get("flour_delivery", {})
 if int(stock.get("bread_loaf", company.get("packed_bread_count", "0"))) >= 24:
  bakery_status.text = "Пекарня · хлеб готов к продаже"
 elif not delivery.is_empty() or int(company.get("mill_delivery_tick", "0")) > 0:
  var due := int(delivery.get("delivery_tick", company.get("mill_delivery_tick", "0")))
  bakery_status.text = "Пекарня · мука в пути до тика %s" % due
 elif int(stock.get("wheat_flour", company.get("flour_g", "0"))) >= 900:
  bakery_status.text = "Пекарня · мука на складе"
 else:
  bakery_status.text = "Пекарня · ждёт закупки"
 var sold := int(company.get("completed_batches", "0"))
 if sold > 0:
  bakery_status.text += "\nПродано партий: %s" % sold

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
 canvas.resized.connect(_layout_panel)
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
 _layout_panel()
 canvas.add_child(panel)
 panel.hide()
 advance_status()

func _layout_panel() -> void:
 if panel == null or canvas == null:
  return
 var usable := canvas.size
 var width := minf(800.0, maxf(240.0, usable.x - 32.0))
 var height := minf(510.0, maxf(180.0, usable.y - 112.0))
 panel.offset_left = -width / 2.0
 panel.offset_right = width / 2.0
 panel.offset_top = -height / 2.0
 panel.offset_bottom = height / 2.0

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
  button(name, func():
   city.visible = true
   panel.hide()
   status.text = "  " + name + " • демонстрационная сцена", v)
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
 var overview := "Баланс: %s ₽\nПредприятий: 1\n" % cash
 overview += "Молочный завод • загрузка 72%%\nПоследний демотик: %s" % tick
 label(overview, v)
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
 var cell_info := "  Выбрана ячейка %s\n  %s\n" % [selected + 1, roles[selected]]
 cell_info += "  Качество: 38 / 100\n  Загрузка: 72%%"
 label(cell_info, details)
 button("Сменить продукт", func():
  roles[selected] = "Коктейль" if roles[selected] != "Коктейль" else "Розлив"
  show_factory(), details)
 button("Начать связь",func(): linked=selected;status.text="  Выберите ячейку-получатель",details)
 button("Модернизация −5000 ₽",func():
  if cash>=5000:
   cash-=5000
   advance_status(),details)
 label("Деморецепты и показатели рассчитаны локально в прототипе.",v,15)
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
 if fps == null:
  return
 fps.text=" %s FPS " % Engine.get_frames_per_second()

func _unhandled_input(event: InputEvent) -> void:
 if event is InputEventScreenTouch:
  if event.pressed:
   touches[event.index] = event.position
   if touches.size() == 1:
    press_position = event.position
    touch_moved = false
   else:
    touch_moved = true
  else:
   if touches.size() == 1 and not touch_moved and city.visible:
    if panel != null:
     show_factory()
    else:
     pick_lot(event.position)
   touches.erase(event.index)
  pinch_distance = _touch_distance() if touches.size() == 2 else 0.0
 elif event is InputEventScreenDrag:
  if not touches.has(event.index):
   return
  var previous: Vector2 = touches[event.index]
  touches[event.index] = event.position
  if event.position.distance_to(press_position) > 6.0:
   touch_moved = true
  if touches.size() == 2:
   var distance := _touch_distance()
   if pinch_distance > 0.0 and distance > 0.0:
    camera.size = clampf(camera.size * pinch_distance / distance, 22.0, 110.0)
   pinch_distance = distance
  elif touches.size() == 1:
   _pan_camera(event.position - previous)
 elif event is InputEventMagnifyGesture:
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
    press_position=event.position
    moved=false
   elif not moved and city.visible:
    if panel != null:
     show_factory()
    else:
     pick_lot(event.position)
 elif event is InputEventMouseMotion and dragging:
  var d: Vector2=event.position-last
  last=event.position
  moved=moved or event.position.distance_to(press_position)>6
  _pan_camera(d)

func _touch_distance() -> float:
 var ids := touches.keys()
 if ids.size() != 2:
  return 0.0
 return (touches[ids[0]] as Vector2).distance_to(touches[ids[1]])

func _pan_camera(delta: Vector2) -> void:
 camera.position += Vector3(-delta.x - delta.y, 0, delta.x - delta.y) * camera.size / 900.0 * (get_node("/root/Preferences").pan_speed / 100.0)
