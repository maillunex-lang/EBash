#!/usr/bin/env python3
"""Build read-only encyclopedia JSON from the actual EBash catalogs; never activate drafts."""
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CAT = ROOT / 'docs' / 'catalogs'
OUT = ROOT / 'web' / 'wiki' / 'catalog.json'

FAMILIES = {
 'grain':'Зерно и хлеб','dairy':'Молочные продукты','wood':'Древесина и бумага',
 'vegetables':'Овощи и фрукты','oil':'Нефть и полимеры','construction':'Строительство',
 'glass':'Стекольная отрасль'
}
# These are interface translations, not production-balance assumptions.
NAMES = '''
barley_field|Ячменное поле
barley_flour|Ячменная мука
barley_grain|Зерно ячменя
bitumen|Битум
bottle_cap|Крышка бутылки
bread_dough|Хлебное тесто
bread_loaf|Буханка хлеба
bread_pack|Упакованный хлеб
brick|Кирпич
butter|Сливочное масло
cardboard|Картон
clay|Глина
clay_deposit|Месторождение глины
cocoa_drink|Какао-напиток
cocoa_drink_bottle|Какао в бутылке
cocoa_powder|Какао-порошок
concrete|Бетон
cream|Сливки
crude_oil|Сырая нефть
dairy_farm|Молочная ферма
energy|Энергия
forest|Лесной участок
glass_bottle|Стеклянная бутылка
glass_jar|Стеклянная банка
glass_melt|Стекломасса
gravel|Гравий
limestone|Известняк
limestone_deposit|Месторождение известняка
logs|Брёвна
milk_bottle|Молоко в бутылке
oil_deposit|Нефтяное месторождение
paper|Бумага
paper_pack|Упакованная бумага
pasteurized_milk|Пастеризованное молоко
plastic_bottle|Пластиковая бутылка
plastic_resin|Полимерная смола
potato|Картофель
potato_field|Картофельное поле
potato_pack|Упакованный картофель
quartz_sand|Кварцевый песок
raw_milk|Сырое молоко
rebar|Арматура
refined_fuel|Очищенное топливо
reinforced_concrete|Железобетон
sand_deposit|Месторождение песка
sawn_timber|Пиломатериалы
soda_ash|Кальцинированная сода
soda_ash_supply|Поставка кальцинированной соды
steel_supply|Поставка стали
stone_deposit|Каменный карьер
sunflower_field|Поле подсолнечника
sunflower_oil|Подсолнечное масло
sunflower_oil_bottle|Масло подсолнечное в бутылке
sunflower_seed|Семена подсолнечника
washed_potato|Вымытый картофель
water|Вода
wheat_field|Пшеничное поле
wheat_flour|Пшеничная мука
wheat_grain|Зерно пшеницы
wood_pulp|Древесная целлюлоза
yeast|Дрожжи
rye_field|Ржаное поле
rye_grain|Зерно ржи
rye_flour|Ржаная мука
rye_dough|Ржаное тесто
rye_bread|Ржаной хлеб
oat_field|Овсяное поле
oat_grain|Зерно овса
oat_flakes|Овсяные хлопья
wheat_bran|Пшеничные отруби
animal_feed|Корм для животных
pasta_dough|Тесто для макарон
dry_pasta|Сухие макароны
skimmed_milk|Обезжиренное молоко
milk_powder|Сухое молоко
starter_culture|Закваска
yogurt_base|Основа йогурта
yogurt|Йогурт
cheese_curd|Сырное зерно
cheese|Сыр
whey|Сыворотка
sour_cream|Сметана
sugar|Сахар
ice_cream_mix|Смесь для мороженого
ice_cream|Мороженое
wood_chips|Древесная щепа
resin_binder|Связующая смола
particle_board|ДСП
plywood|Фанера
wood_pellets|Древесные пеллеты
paper_roll|Рулон бумаги
paper_bag|Бумажный пакет
corrugated_board|Гофрокартон
carton_box|Картонная коробка
carrot_field|Морковное поле
carrot|Морковь
washed_carrot|Вымытая морковь
carrot_pack|Упакованная морковь
onion_field|Луковое поле
onion|Лук
onion_pack|Упакованный лук
tomato_greenhouse|Теплица с томатами
tomato|Томат
tomato_paste|Томатная паста
tomato_sauce|Томатный соус
apple_orchard|Яблоневый сад
apple|Яблоко
apple_juice|Яблочный сок
apple_juice_bottle|Яблочный сок в бутылке
potato_starch|Картофельный крахмал
diesel_fraction|Дизельная фракция
gasoline_fraction|Бензиновая фракция
lubricant_base|Основа смазки
lubricant_pack|Упакованная смазка
plastic_film|Пластиковая плёнка
plastic_container|Пластиковая тара
plastic_pipe|Пластиковая труба
plastic_packaging|Полимерная упаковка
cement_clinker|Цементный клинкер
cement|Цемент
concrete_block|Бетонный блок
paving_slab|Тротуарная плита
ceramic_tile|Керамическая плитка
roof_tile|Кровельная черепица
sand_supply|Поставка песка
dry_mortar|Сухая строительная смесь
mortar_mix|Строительный раствор
concrete_panel|Бетонная панель
frame_supply|Поставка каркасов
glass_window|Оконное стекло
glass_sheet|Листовое стекло
glass_fiber|Стекловолокно
glass_cullet|Стеклобой
recycled_glass_melt|Переплавленное стекло
glassware|Изделия из стекла
glass_packaging|Стеклянная упаковка
salt|Соль
buttermilk|Пахта
gypsum|Гипс
'''
NAMES = dict(line.split('|', 1) for line in NAMES.strip().splitlines())

def load(name):
 return json.loads((CAT / name).read_text(encoding='utf-8'))

def parts(entries, include_quantities=True):
 # Draft input values are not authorized game balances. Display structure only.
 return [{'good':x['good_id'], 'quantity':x.get('quantity') if include_quantities else None,
          'unit':x.get('unit') if include_quantities else None} for x in entries]

def main():
 raw_goods=load('resources.seed.json')['resources']
 raw_draft=load('recipes.seed.json')['recipes']
 approved=load('game-production-profiles.v1.json')
 assert approved['status']=='approved_game_balance_v1'
 assert len(raw_goods)==len({x['id'] for x in raw_goods})
 assert len(raw_draft)==len({x['id'] for x in raw_draft})
 assert all(x['status']=='draft_not_executable' for x in raw_draft)
 goods=[{'id':x['id'],'name':NAMES.get(x['id'],x['id'].replace('_',' ').capitalize()),'kind':x['kind']} for x in raw_goods]
 assert len(NAMES)==len(raw_goods) and {x['id'] for x in raw_goods}==set(NAMES), 'Missing or extra resource localization'
 recipes=[]
 for x in raw_draft:
  outs=parts(x['outputs'], include_quantities=False)
  title='Получение: '+', '.join(NAMES.get(y['good'],y['good']) for y in outs[:2])
  recipes.append({'id':'draft:'+x['id'],'source_id':x['id'],'name':title,'family':x['family'],
                  'status':'draft','inputs':parts(x['inputs'], include_quantities=False),'outputs':outs,'version':x['version']})
 for x in approved['profiles']:
  outs=[{'good':k,'quantity':v,'unit':'шт.' if k=='bread_loaf' else 'г'} for k,v in x['batch_outputs'].items()]
  ins=[{'good':k,'quantity':v,'unit':'г'} for k,v in x['batch_inputs'].items()]
  name='Производство: '+', '.join(NAMES.get(y['good'],y['good']) for y in outs)
  recipes.append({'id':'approved:'+x['id'],'source_id':x['recipe_id'],'name':name,
                  'family':'grain','status':'approved','inputs':ins,'outputs':outs,
                  'version':x['recipe_version'],'equipment':x['equipment_type'],
                  'energy_wh':x['energy_wh_per_batch'],'minimum_batches':x['minimum_batches'],
                  'workers':x['workers_per_active_line'],'cost_kopecks':x['game_operation_cost_kopecks'],
                  'notes':x.get('notes','')})
 referenced={v['good'] for x in recipes for side in ('inputs','outputs') for v in x[side]}
 assert referenced.issubset({x['id'] for x in goods}), f'Unregistered goods: {referenced-{x["id"] for x in goods}}'
 OUT.parent.mkdir(parents=True,exist_ok=True)
 OUT.write_text(json.dumps({'schema_version':1,'source':'EBash DES-004 + approved game profiles v1',
                            'families':FAMILIES,'goods':goods,'recipes':recipes},ensure_ascii=False,separators=(',',':'))+'\n',encoding='utf-8')
 print(f'Wiki catalog: {len(goods)} goods, {len(recipes)} recipes, {len(approved["profiles"])} approved, {len(raw_draft)} drafts -> {OUT}')

if __name__=='__main__': main()
