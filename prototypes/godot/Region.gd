extends Control

const Catalog := preload("res://CityCatalog.gd")
const Client := preload("res://PilotClient.gd")
var body := VBoxContainer.new()
var title := Label.new()
var api := Client.new()
var region_tab: Button
var company_tab: Button
var city_buttons: Dictionary = {}

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

func show_company() -> void:
 page("Моя компания",true)
 var summary := Label.new()
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
 var data: Dictionary = response.data
 var build: Dictionary = data.get("game_production",{}).get("construction",{})
 var facility := "Пекарня · Уфа"
 if not build.is_empty() and not build.open:
  facility = "Свободен выбор участка" if build.site == "" else "Пекарня строится · Уфа"
 summary.text = "Баланс   %.2f ₽\nИгровой тик   %s\n%s\nПродано партий   %s" % [
  int(data.cash_kopecks)/100.0,data.last_committed_tick,facility,data.get("completed_batches","0")]
