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
var dragging := false
var touches: Dictionary = {}
var pinch_distance := 0.0
var touch_moved := false
var press_position := Vector2.ZERO
var last := Vector2.ZERO
var moved := false

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
 # D-192: только игровой маршрут города, локальный UI-стенд не используется.
 get_tree().set_meta("ebash_city_route",true)
 city_id = str(get_tree().get_meta("ebash_city_id","ufa"))
 get_tree().set_meta("ebash_city_id",city_id)
 city = Node3D.new()
 add_child(city)
 build_district()
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
 make_game_city()

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
   if touches.size() == 1 and not touch_moved:
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
   elif not moved:
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
