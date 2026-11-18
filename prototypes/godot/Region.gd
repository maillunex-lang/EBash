extends Control

const Catalog := preload("res://CityCatalog.gd")
const Client := preload("res://PilotClient.gd")
var body := VBoxContainer.new()
var title := Label.new()
var api := Client.new()
var region_tab: Button
var company_tab: Button
var city_buttons: Dictionary = {}
var enterprise_buttons: Dictionary = {}
var company_data: Dictionary = {}
var summary: Label

func action(parent: Container, caption: String, callback: Callable) -> Button:
 var item := Button.new()
 item.text = caption
 item.custom_minimum_size = Vector2(200,58)
 item.add_theme_font_size_override("font_size",21)
 parent.add_child(item)
 item.pressed.connect(callback)
 return item

func _ready() -> void:
 add_child(api)
 var background := ColorRect.new()
 background.color = Color("182a31")
 background.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
 add_child(background)
 var margin := MarginContainer.new()
 margin.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
 for side in ["left","right","top","bottom"]:
  margin.add_theme_constant_override("margin_"+side,28)
 add_child(margin)
 var row := HBoxContainer.new()
 row.add_theme_constant_override("separation",38)
 margin.add_child(row)
 var rail := VBoxContainer.new()
 row.add_child(rail)
 region_tab = action(rail,"Республика",show_region)
 company_tab = action(rail,"Моя компания",show_company)
 action(rail,"Главное меню",func(): get_tree().change_scene_to_file("res://Launcher.tscn"))
 var column := VBoxContainer.new()
 row.add_child(column)
 column.add_theme_constant_override("separation",20)
 title.add_theme_font_size_override("font_size",32)
 column.add_child(title)
 column.add_child(body)
 body.add_theme_constant_override("separation",12)
 if get_tree().get_meta("ebash_hub_page","region") == "company":
  show_company()
 else:
  show_region()

func page(caption: String, company: bool) -> void:
 get_tree().set_meta("ebash_hub_page","company" if company else "region")
 title.text = caption
 company_tab.disabled = company
 region_tab.disabled = not company
 city_buttons.clear()
 enterprise_buttons.clear()
 summary = null
 for child in body.get_children():
  body.remove_child(child)
  child.queue_free()

func show_region() -> void:
 page("Республика Башкортостан",false)
 for id: String in Catalog.CITIES:
  city_buttons[id] = action(body,Catalog.CITIES[id].title,open_city.bind(id))

func open_city(id: String) -> void:
 get_tree().set_meta("ebash_city_id",id)
 get_tree().set_meta("ebash_city_route",true)
 get_tree().change_scene_to_file("res://Main.tscn")

func open_site(lot_id: String) -> void:
 get_tree().set_meta("ebash_focus_lot",int(lot_id))
 open_city("ufa")

func open_manage() -> void:
 get_tree().set_meta("ebash_city_id","ufa")
 get_tree().set_meta("ebash_city_route",true)
 get_tree().change_scene_to_file("res://Pilot.tscn")

func show_company() -> void:
 page("Моя компания",true)
 summary = Label.new()
 summary.add_theme_font_size_override("font_size",22)
 body.add_child(summary)
 action(body,"Перейти в Уфу",open_city.bind("ufa"))
 if not api.configure_from_session():
  summary.text = "Для просмотра компании подключитесь\nк серверу через карту Уфы."
  return
 summary.text = "Загружаем данные компании…"
 var response := await api.fetch("/company")
 if not is_instance_valid(summary) or not summary.is_inside_tree():
  return
 if response.status != 200:
  summary.text = "Не удалось загрузить данные компании."
  action(body,"Повторить",show_company)
  return
 fill_company(response.data)

func fill_company(data: Dictionary) -> void:
 company_data = data
 enterprise_buttons.clear()
 var keep: Array[Node] = []
 if summary != null and is_instance_valid(summary):
  keep.append(summary)
 for child in body.get_children():
  if child in keep:
   continue
  body.remove_child(child)
  child.queue_free()
 if summary == null or not is_instance_valid(summary):
  summary = Label.new()
  summary.add_theme_font_size_override("font_size",22)
  body.add_child(summary)
 var build: Dictionary = data.get("game_production",{}).get("construction",{})
 var cash := int(data.get("cash_kopecks",0))
 var tick := int(data.get("last_committed_tick",0))
 var sold := str(data.get("completed_batches","0"))
 summary.text = "Баланс   %.2f ₽\nИгровой тик   %s\nПродано партий   %s" % [cash/100.0,tick,sold]
 _fill_enterprises(build,tick)
 action(body,"Рынок",show_market)
 action(body,"Финансы",show_finance)
 action(body,"Перейти в Уфу",open_city.bind("ufa"))

func _fill_enterprises(build: Dictionary, tick: int) -> void:
 var heading := Label.new()
 heading.text = "Предприятия"
 heading.add_theme_font_size_override("font_size",24)
 body.add_child(heading)
 if build.is_empty():
  var empty := Label.new()
  empty.text = "Нет данных о площадках. Откройте карту Уфы."
  empty.add_theme_font_size_override("font_size",20)
  body.add_child(empty)
  return
 if bool(build.get("open",false)):
  var site := str(build.get("site",""))
  var open_label := Label.new()
  open_label.text = "Пекарня · Уфа · участок %s" % site
  open_label.add_theme_font_size_override("font_size",20)
  body.add_child(open_label)
  enterprise_buttons["manage"] = action(body,"Управление",open_manage)
  if site != "":
   enterprise_buttons["map"] = action(body,"Показать на карте",open_site.bind(site))
  return
 var site_id := str(build.get("site",""))
 var duration := int(build.get("duration",0))
 if site_id == "":
  var pick := Label.new()
  pick.text = "Свободен выбор участка · срок стройки %s тика" % duration
  pick.add_theme_font_size_override("font_size",20)
  body.add_child(pick)
  var sites: Dictionary = build.get("sites",{})
  var ids: Array = sites.keys()
  ids.sort()
  for id in ids:
   var price := int(sites[id])
   var caption := "Участок %s · %s ₽ · %s тика" % [id,price/100,duration]
   enterprise_buttons[str(id)] = action(body,caption,open_site.bind(str(id)))
  return
 var ready := int(build.get("ready_tick",0))
 var left: int = maxi(ready-tick,0)
 var works := Label.new()
 works.text = "Пекарня строится · Уфа · участок %s\nПриёмка на тике %s · осталось %s тика" % [site_id,ready,left]
 works.add_theme_font_size_override("font_size",20)
 body.add_child(works)
 enterprise_buttons["works"] = action(body,"Открыть стройку",open_site.bind(site_id))

func show_market() -> void:
 page("Рынок",true)
 var note := Label.new()
 note.add_theme_font_size_override("font_size",22)
 body.add_child(note)
 if company_data.is_empty():
  note.text = "Сначала загрузите данные компании."
  action(body,"К компании",show_company)
  return
 var production: Dictionary = company_data.get("game_production",{})
 var stock: Dictionary = production.get("stock",{})
 var delivery: Dictionary = production.get("flour_delivery",{})
 var flour := int(stock.get("wheat_flour",company_data.get("flour_g",0)))
 var bread := int(stock.get("bread_loaf",company_data.get("packed_bread_count",0)))
 var lines: PackedStringArray = PackedStringArray([
  "Цены и наличие подтверждает сервер.",
  "Мука на складе: %s г" % flour,
  "Хлеб на складе: %s шт" % bread,
 ])
 if not delivery.is_empty() or int(company_data.get("mill_delivery_tick",0)) > 0:
  var due := int(delivery.get("delivery_tick",company_data.get("mill_delivery_tick",0)))
  lines.append("Поставка муки до тика %s" % due)
 note.text = "\n".join(lines)
 action(body,"К закупке муки",open_manage)
 action(body,"К компании",show_company)

func show_finance() -> void:
 page("Финансы",true)
 var note := Label.new()
 note.add_theme_font_size_override("font_size",22)
 body.add_child(note)
 if company_data.is_empty():
  note.text = "Сначала загрузите данные компании."
  action(body,"К компании",show_company)
  return
 var cash := int(company_data.get("cash_kopecks",0))
 var revenue := int(company_data.get("revenue_kopecks",0))
 var cost := int(company_data.get("cost_kopecks",0))
 var profit := int(company_data.get("profit_kopecks",0))
 var tick := int(company_data.get("last_committed_tick",0))
 var sold := str(company_data.get("completed_batches","0"))
 note.text = "Доступно   %.2f ₽\nТик   %s\nВыручка   %.2f ₽\nЗатраты   %.2f ₽\nПрибыль   %.2f ₽\nПродано партий   %s" % [
  cash/100.0,tick,revenue/100.0,cost/100.0,profit/100.0,sold]
 action(body,"К компании",show_company)
