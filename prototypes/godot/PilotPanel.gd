extends Control

const PilotClient := preload("res://PilotClient.gd")
const ORDER_ID := "godot-pilot-buy-v1"
const LINE_ID := "godot-pilot-line-v1"

const GOODS := {
 "raw_milk_g": "Сырое молоко", "cream_g": "Сливки", "skimmed_milk_g": "Обезжиренное молоко",
 "logs_g": "Брёвна", "wood_chips_g": "Щепа", "wood_pellets_g": "Пеллеты",
 "quartz_sand_g": "Кварцевый песок", "limestone_g": "Известняк", "soda_ash_g": "Кальцинированная сода",
 "glass_melt_g": "Стекломасса", "glass_jar_count": "Стеклянные банки",
 "crude_oil_g": "Сырая нефть", "lubricant_base_g": "Базовое масло", "lubricant_pack_count": "Фасованное масло",
 "plastic_bottle_count": "Пластиковые бутылки", "bottle_cap_count": "Крышки",
 "tomato_g": "Томаты", "tomato_paste_g": "Томатная паста", "cement_g": "Цемент",
 "sand_supply_g": "Песок", "water_g": "Вода", "dry_mortar_g": "Сухая смесь", "mortar_mix_g": "Раствор",
}

var api := PilotClient.new()
var snapshot: Dictionary = {}
var mill_offer: Dictionary = {}
var supplies: Array = []
var supply_states: Dictionary = {}
var supply_buttons: VBoxContainer
var remote_offer: Dictionary = {}
var remote_order_state: Dictionary = {}
var remote_buy_button: Button
var mill_order_state: Dictionary = {}
var mill_grain_state: Dictionary = {}
var mill_list_state: Dictionary = {}
var mill_grain_button: Button
var mill_list_button: Button
var order_state: Dictionary = {}
var line_state: Dictionary = {}
var sale_state: Dictionary = {}
var last_sale_id: String = ""
var configured := false
var busy := false
var fresh := false
var recovering := false
var message: Label
var inventory: Label
var chain_picker: OptionButton
var chains: Array = []
var command_log: Label
var refresh_button: Button
var tick_button: Button
var demo_hint: Label
var demo := false
var buy_button: Button
var mill_buy_button: Button
var mill_offer_label: Label
var line_button: Button
var sell_button: Button

func _ready() -> void:
 add_child(api)
 make_interface()
 configured = api.configure(OS.get_environment("EBASH_PILOT_API"),
  OS.get_environment("EBASH_PILOT_WORLD"), OS.get_environment("EBASH_PILOT_TOKEN"))
 if not configured:
  message.text = "Укажите EBASH_PILOT_API, EBASH_PILOT_WORLD и EBASH_PILOT_TOKEN."
 update_controls()
 if configured:
  await refresh()

func load_chains() -> void:
 var response := await api.fetch("/pilot-chains")
 if response["status"] == 200:
  chains = response["data"].get("chains", [])
  for chain: Dictionary in chains:
   chain_picker.add_item(chain["label"])

func selected_chain() -> Dictionary:
 var batch: Dictionary = snapshot.get("chain_batch", {})
 if not batch.is_empty():
  for chain: Dictionary in chains:
   if chain["id"] == batch["id"]:
    return chain
 if chain_picker.selected > 0:
  return chains[chain_picker.selected - 1]
 return {}

func make_interface() -> void:
 var background := ColorRect.new()
 background.color = Color("101923")
 background.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
 add_child(background)
 var margin := MarginContainer.new()
 margin.set_anchors_and_offsets_preset(Control.PRESET_FULL_RECT)
 for side: String in ["left", "top", "right", "bottom"]:
  margin.add_theme_constant_override("margin_" + side, 32)
 add_child(margin)
 var scroll := ScrollContainer.new()
 scroll.horizontal_scroll_mode = ScrollContainer.SCROLL_MODE_DISABLED
 margin.add_child(scroll)
 var column := VBoxContainer.new()
 column.size_flags_horizontal = Control.SIZE_EXPAND_FILL
 column.add_theme_constant_override("separation", 14)
 scroll.add_child(column)
 add_label(column, "Производственные цепочки · тестовый режим", 30)
 chain_picker = OptionButton.new()
 chain_picker.add_item("Пшеница → упакованный хлеб")
 chain_picker.item_selected.connect(func(_index: int) -> void: update_controls())
 column.add_child(chain_picker)
 inventory = add_label(column, "Состояние сервера ещё не получено", 22)
 demo_hint = add_label(column, "", 20)
 tick_button = add_button(column, "Следующий тик", advance_demo_tick)
 refresh_button = add_button(column, "Обновить состояние", refresh)
 buy_button = add_button(column, "Закупить контрольную партию с упаковкой · 504 ₽", order_inputs)
 mill_offer_label = add_label(column, "", 18)
 mill_buy_button = add_button(column, "Купить муку у мельницы", order_mill_flour)
 remote_buy_button = add_button(column, "", order_remote_flour)
 mill_grain_button = add_button(column, "Купить 10 кг зерна · 300 ₽", order_mill_grain)
 mill_list_button = add_button(column, "Выставить 7,4 кг муки", list_mill_flour)
 supply_buttons = VBoxContainer.new()
 column.add_child(supply_buttons)
 line_button = add_button(column, "Настроить тестовую линию", configure_line)
 sell_button = add_button(column, "Продать партию · 1 080 ₽", sell_bread)
 command_log = add_label(column, "", 18)
 message = add_label(column, "", 18)

func add_label(parent: Control, value: String, font_size: int) -> Label:
 var item := Label.new()
 item.text = value
 item.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
 item.add_theme_font_size_override("font_size", font_size)
 parent.add_child(item)
 return item

func add_button(parent: Control, title: String, action: Callable) -> Button:
 var item := Button.new()
 item.text = title
 item.custom_minimum_size.y = 48
 item.add_theme_font_size_override("font_size", 20)
 item.pressed.connect(action)
 parent.add_child(item)
 return item

func update_controls() -> void:
 refresh_button.disabled = busy or not configured
 var blocked := busy or not fresh or recovering
 var seller: bool = mill_offer.get("seller", false)
 var purchasing: bool = has_pending_supply() or mill_order_state.get("status") == "accepted" or remote_order_state.get("status") == "accepted"
 buy_button.disabled = blocked or batch_active() or has_command(order_state) or purchasing
 buy_button.visible = not seller and not demo
 chain_picker.visible = not seller and not demo
 tick_button.visible = demo
 tick_button.disabled = blocked
 demo_hint.visible = demo
 line_button.visible = not seller
 sell_button.visible = not seller
 mill_grain_button.visible = seller
 mill_list_button.visible = seller
 mill_grain_button.disabled = blocked or has_command(mill_grain_state) or str(snapshot.get("mill_grain_g", "0")) != "0" or str(snapshot.get("mill_flour_g", "0")) != "0"
 mill_list_button.disabled = blocked or has_command(mill_list_state) or not snapshot.get("mill_unlisted", false)
 chain_picker.disabled = busy or purchasing or has_command(order_state) or str(snapshot.get("grain_ready_tick", "0")) != "0" or snapshot.has("chain_batch")
 var chain := selected_chain()
 mill_offer_label.visible = not seller and chain.is_empty() and not mill_offer.is_empty()
 remote_buy_button.visible = not seller and chain.is_empty() and remote_offer.get("available", false)
 var mill_blocked: bool = blocked or purchasing or order_state.get("status") == "accepted" or snapshot.has("next_mill_order")
 remote_buy_button.disabled = mill_blocked or has_command(remote_order_state)
 if remote_buy_button.visible:
  var total := str(remote_offer["price_kopecks"]).to_int() + str(remote_offer["ingredients_kopecks"]).to_int()
  remote_buy_button.text = "Дальняя мельница · 7,4 кг за %s ₽ · доставка через %s тика · всего %s ₽" % [
   format_money(remote_offer["price_kopecks"]), remote_offer["delivery_ticks"], format_money(str(total))]
 mill_buy_button.visible = mill_offer_label.visible
 mill_buy_button.disabled = mill_blocked or has_command(mill_order_state) or not mill_offer.get("available", false)
 mill_buy_button.text = "Купить муку про запас" if batch_active() else "Купить муку у мельницы"
 if remote_buy_button.visible and batch_active():
  remote_buy_button.text = "Про запас · " + remote_buy_button.text
 if mill_offer_label.visible:
  if mill_offer.get("available", false):
   var flour_cost := str(mill_offer["price_kopecks"]).to_int()
   var supplies_cost := str(mill_offer["ingredients_kopecks"]).to_int()
   mill_offer_label.text = "Ближняя мельница: %s г муки за %s ₽. Доставка в тик подтверждения. Ингредиенты и упаковка — %s ₽. Всего %s ₽." % [
    mill_offer["quantity_g"], format_money(str(flour_cost)), format_money(str(supplies_cost)),
    format_money(str(flour_cost + supplies_cost))]
  else:
   mill_offer_label.text = "Мельница пока не отгружает муку."
 buy_button.text = "Закупить контрольную партию с упаковкой · 504 ₽" if chain.is_empty() else "Закупить партию · %s ₽" % (str(chain["cost"]).to_int() / 100)
 sell_button.text = "Продать партию · 1 080 ₽" if chain.is_empty() else "Продать %s × %s коп." % [chain["quantity"], chain["price"]]
 var current_batch: Dictionary = snapshot.get("chain_batch", {})
 var advance := batch_active() and not current_batch.is_empty()
 show_supply_choices(chain, blocked or purchasing or order_state.get("status") == "accepted" or current_batch.has("next_order"), advance)
 line_button.disabled = blocked or has_command(line_state)
 sell_button.disabled = blocked or str(snapshot.get("bread_lot_id", "")).is_empty() or has_command(sale_state)

func has_pending_supply() -> bool:
 for state: Dictionary in supply_states.values():
  if state.get("status") == "accepted" or state.get("status") == "unknown":
   return true
 return false

func show_supply_choices(chain: Dictionary, blocked: bool, advance: bool) -> void:
 for child: Node in supply_buttons.get_children():
  supply_buttons.remove_child(child)
  child.queue_free()
 for offer: Dictionary in supplies:
  if offer["chain"] != chain.get("id") or not offer["available"]:
   continue
  var delay := int(offer["delivery_ticks"])
  var timing := "в тик сделки" if delay == 0 else "через %s тика" % delay
  var title := "%s · %s ₽ · доставка %s" % [offer["label"], format_money(offer["price_kopecks"]), timing]
  var button := add_button(supply_buttons, ("Заказать про запас · " if advance else "") + title, order_supply.bind(offer))
  button.disabled = blocked

func supply_order_id(offer: Dictionary) -> String:
 return order_id() + "-supply-" + str(offer["offer_id"])

func order_supply(offer: Dictionary) -> void:
 await send_command(supply_order_id(offer), "buy_chain_supply", {
  "offer_id": offer["offer_id"], "quantity": "1", "max_price_kopecks": offer["price_kopecks"],
 })

func batch_active() -> bool:
 var started := str(snapshot.get("grain_ready_tick", "0")) != "0" or snapshot.has("chain_batch")
 return started and (str(snapshot.get("bread_ready_tick", "0")) == "0" or not str(snapshot.get("bread_lot_id", "")).is_empty())

func order_id() -> String:
 var completed := str(snapshot.get("completed_batches", "0")).to_int()
 if completed == 0 and not batch_active() and str(snapshot.get("bread_ready_tick", "0")) != "0":
  completed = 1
 return ORDER_ID if completed == 0 else ORDER_ID + "-" + str(completed)

func has_command(state: Dictionary) -> bool:
 return state.get("status", "") in ["accepted", "applied", "rejected"]

func command_state(id: String) -> Dictionary:
 var response := await api.fetch("/commands/" + id)
 if response["status"] == 200:
  return response["data"]
 if response["status"] == 404:
  return {}
 return {"status": "unknown"}

func refresh() -> void:
 if not configured or busy:
  return
 busy = true
 update_controls()
 message.text = "Запрос состояния…"
 if chains.is_empty():
  await load_chains()
 var result := await api.fetch("/company")
 if result["status"] == 200:
  snapshot = result["data"]
  var batch: Dictionary = snapshot.get("chain_batch", {})
  for index: int in range(chains.size()):
   if chains[index]["id"] == batch.get("id", ""):
    chain_picker.select(index + 1)
  order_state = await command_state(order_id())
  line_state = await command_state(LINE_ID)
  var lot := str(snapshot.get("bread_lot_id", ""))
  if not lot.is_empty():
   last_sale_id = sale_id(lot)
  elif str(snapshot.get("bread_ready_tick", "0")).to_int() > 0:
   var production_tick := str(snapshot["bread_ready_tick"]).to_int() - 1
   last_sale_id = sale_id("bread-" + str(production_tick))
  else:
   last_sale_id = ""
  sale_state = {} if last_sale_id.is_empty() else await command_state(last_sale_id)
  var offer := await api.fetch("/mill-offer")
  mill_offer = offer["data"] if offer["status"] == 200 else {}
  mill_order_state = {} if mill_offer.is_empty() else await command_state(mill_order_id())
  if not mill_offer.get("seller", false):
   var remote := await api.fetch("/mill-offer?supplier=remote")
   remote_offer = remote["data"] if remote["status"] == 200 else {}
   remote_order_state = {} if remote_offer.is_empty() else await command_state(remote_order_id())
  if mill_offer.get("seller", false):
   mill_grain_state = await command_state(mill_action_id("grain"))
   mill_list_state = await command_state(mill_action_id("list"))
  var supply_response := await api.fetch("/chain-supplies")
  supplies = supply_response["data"].get("offers", []) if supply_response["status"] == 200 else []
  supply_states.clear()
  for supply: Dictionary in supplies:
   supply_states[supply["offer_id"]] = await command_state(supply_order_id(supply))
  var world_state := await api.fetch("/status")
  demo = world_state["data"].get("demo", false)
  recovering = world_state["data"].get("mode") == "recovering"
  fresh = world_state["status"] == 200 and order_state.get("status") != "unknown" and line_state.get("status") != "unknown" and sale_state.get("status") != "unknown" and mill_order_state.get("status") != "unknown" and remote_order_state.get("status") != "unknown" and mill_grain_state.get("status") != "unknown" and mill_list_state.get("status") != "unknown"
  show_snapshot()
  if not fresh:
   message.text = "Не удалось получить статусы сервера. Обновите состояние."
  elif recovering:
   message.text = "Восстановление экономики: доступен просмотр."
  else:
   message.text = "Сервер подтвердил состояние."
 else:
  fresh = false
  message.text = "Состояние не обновлено. Код HTTP: %s. Предыдущие данные могут быть устаревшими." % result["status"]
 busy = false
 update_controls()

func show_snapshot() -> void:
 if demo:
  var batches := int(snapshot.get("completed_batches", "0"))
  if batches >= 2:
   demo_hint.text = "Две партии проданы. Выручка %s ₽, прибыль %s ₽." % [money("revenue_kopecks"), money("profit_kopecks")]
  elif not snapshot.get("line_enabled", false):
   demo_hint.text = "Настройте линию и выберите мельницу. Затем нажмите «Следующий тик»."
  elif not str(snapshot.get("bread_lot_id", "")).is_empty():
   demo_hint.text = "Предложите хлеб к продаже. Сделка пройдёт на следующем тике."
  elif batch_active():
   demo_hint.text = "Проведите следующий тик. Если есть свободный лот, можно заказать муку заранее."
  else:
   demo_hint.text = "Выберите мельницу. Ближняя поставляет сразу, дальняя экономит 60 ₽."
 inventory.text = "Тик %s  ·  Счёт %s ₽\nЗерно %s г  ·  Мука %s г  ·  Хлеб %s шт.\nБумажная упаковка %s шт.  ·  Упаковано %s шт.\nВыручка %s ₽  ·  Прибыль %s ₽" % [
  snapshot.get("last_committed_tick", "—"), money("cash_kopecks"),
  snapshot.get("grain_g", "—"), snapshot.get("flour_g", "—"), snapshot.get("bread_count", "—"),
  snapshot.get("paper_pack_count", "—"), snapshot.get("packed_bread_count", "—"),
  money("revenue_kopecks"), money("profit_kopecks")]
 if snapshot.get("pack_order", false) and snapshot.get("line_enabled", false) and str(snapshot.get("flour_g", "0")).to_int() >= 7200 and str(snapshot.get("paper_pack_count", "0")).to_int() < 24:
  inventory.text += "\nЛиния ждёт бумажную упаковку. Для партии нужно 24 шт."
 var batch: Dictionary = snapshot.get("chain_batch", {})
 if not batch.is_empty():
  var stock: Dictionary = batch["stock"]
  var goods := PackedStringArray()
  for item: String in stock:
   if int(stock[item]) > 0:
    goods.append("%s — %s %s" % [GOODS.get(item, item), int(stock[item]), "г" if item.ends_with("_g") else "шт."])
  inventory.text = "Тик %s  ·  Счёт %s ₽\nСклад: %s\nВыручка %s ₽  ·  Прибыль %s ₽" % [
   snapshot.get("last_committed_tick", "—"), money("cash_kopecks"),
   " · ".join(goods) if not goods.is_empty() else "пусто",
   money("revenue_kopecks"), money("profit_kopecks")]
 if mill_offer.get("seller", false):
  inventory.text = "Мельница · Тик %s · Счёт %s ₽\nЗерно %s г · Мука %s г · %s" % [
   snapshot.get("last_committed_tick", "—"), money("cash_kopecks"),
   snapshot.get("mill_grain_g", "0"), snapshot.get("mill_flour_g", "0"),
   "Мука ждёт выставления" if snapshot.get("mill_unlisted", false) else "Запас доступен для продажи"]
  command_log.text = "Закупка зерна: %s · Выставление: %s" % [state_text(mill_grain_state), state_text(mill_list_state)]
  return
 if batch.has("next_order"):
  var next_delivery := int(batch["next_order"]["delivery_tick"])
  var arrived := int(snapshot["last_committed_tick"]) >= next_delivery
  inventory.text += "\nСырьё для следующей партии зарезервировано." if arrived else "\nСырьё для следующей партии в пути до тика %s." % next_delivery
 if snapshot.has("next_mill_order"):
  var mill_delivery := int(snapshot["next_mill_order"]["delivery_tick"])
  var ready := int(snapshot["last_committed_tick"]) >= mill_delivery
  inventory.text += "\nМука для следующей партии зарезервирована." if ready else "\nМука для следующей партии в пути до тика %s." % mill_delivery
 var delivery := str(snapshot.get("mill_delivery_tick", "0")).to_int()
 if not batch.is_empty() and int(batch.get("delivery_tick", "0")) > 0:
  var raw_delivery := int(batch["delivery_tick"])
  inventory.text += "\nСырьё в пути. Доставка на тике %s, производство с тика %s." % [raw_delivery, raw_delivery + 1]
 if delivery > 0:
  inventory.text += "\nМука в пути. Доставка на тике %s, доступна для выпечки с тика %s." % [delivery, delivery + 1]
 command_log.text = "Закупка: %s  ·  Линия: %s  ·  Продажа: %s" % [
  state_text(order_state if not order_state.is_empty() else (mill_order_state if not mill_order_state.is_empty() else remote_order_state)), state_text(line_state), state_text(sale_state)]

func money(field: String) -> String:
 return format_money(str(snapshot.get(field, "")))

func format_money(value: String) -> String:
 if not value.is_valid_int():
  return "—"
 var negative := value.begins_with("-")
 if negative:
  value = value.substr(1)
 value = value.pad_zeros(3)
 return ("−" if negative else "") + value.left(-2) + "," + value.right(2)

func state_text(state: Dictionary) -> String:
 match state.get("status", ""):
  "accepted": return "принята на тик %s" % state.get("target_tick", "—")
  "applied": return "выполнена"
  "rejected": return "отклонена (%s)" % state.get("reason", "—")
  "unknown": return "нет статуса"
  _: return "не отправлена"

func order_inputs() -> void:
 var chain := selected_chain()
 await send_command(order_id(), "order_raw",
  {"offer_id": chain.get("id", "pilot-inputs-v2"), "quantity": "1", "max_price_kopecks": chain.get("cost", "50400")})

func mill_order_id() -> String:
 return order_id() + "-mill-" + str(mill_offer.get("offer_id", ""))

func remote_order_id() -> String:
 return order_id() + "-mill-" + str(remote_offer.get("offer_id", ""))

func order_remote_flour() -> void:
 if remote_buy_button.disabled or not remote_buy_button.visible:
  return
 await send_command(remote_order_id(), "buy_mill_flour", {
  "offer_id": remote_offer["offer_id"],
  "quantity": remote_offer["quantity_g"],
  "max_price_kopecks": remote_offer["price_kopecks"],
 })

func mill_action_id(action: String) -> String:
 return "godot-mill-" + action + "-" + str(snapshot.get("last_committed_tick", "0"))

func order_mill_grain() -> void:
 if not mill_grain_button.disabled:
  await send_command(mill_action_id("grain"), "buy_mill_grain", {})

func list_mill_flour() -> void:
 if not mill_list_button.disabled:
  await send_command(mill_action_id("list"), "list_mill_flour", {})

func order_mill_flour() -> void:
 if mill_buy_button.disabled or not mill_buy_button.visible:
  return
 await send_command(mill_order_id(), "buy_mill_flour", {
  "offer_id": mill_offer["offer_id"],
  "quantity": mill_offer["quantity_g"],
  "max_price_kopecks": mill_offer["price_kopecks"],
 })

func configure_line() -> void:
 await send_command(LINE_ID, "configure_line",
  {"enterprise_id": "pilot-bakery", "cell_id": "4", "profile_id": "test-bread-v1"})

func sale_id(lot: String) -> String:
 return "godot-pilot-sale-" + lot

func sell_bread() -> void:
 var lot := str(snapshot.get("bread_lot_id", ""))
 if lot.is_empty():
  return
 last_sale_id = sale_id(lot)
 var chain := selected_chain()
 await send_command(last_sale_id, "offer_output",
  {"lot_id": lot, "quantity": chain.get("quantity", "24"), "price_kopecks": chain.get("price", "4500")})

func advance_demo_tick() -> void:
 if not demo or busy or not fresh or recovering:
  return
 busy = true
 update_controls()
 var next := int(snapshot.get("last_committed_tick", "0")) + 1
 var response := await api.send(HTTPClient.METHOD_POST, "/demo-tick/" + str(next))
 busy = false
 await refresh()
 if response["status"] != 200:
  message.text = "Не удалось провести тик. Проверьте обновлённое состояние."

func send_command(id: String, kind: String, payload: Dictionary) -> void:
 if busy or not fresh or recovering:
  return
 busy = true
 update_controls()
 var result := await api.submit(id, kind, payload)
 busy = false
 if result["status"] == 202:
  await refresh()
 else:
  fresh = false
  update_controls()
  message.text = "Команда не подтверждена (%s). Обновите состояние перед повтором." % result["status"]
