#!/usr/bin/env python3
"""Propose exhaustive, fail-closed sales channels for current draft goods."""
import json
from collections import Counter
from pathlib import Path
root=Path(__file__).resolve().parents[1]/'docs/catalogs'
recipes=json.loads((root/'recipes.seed.json').read_text())['recipes']
produced={o['good_id'] for r in recipes for o in r['outputs']}
consumed={i['good_id'] for r in recipes for i in r['inputs'] if i['input_kind'] in ('produced_good','external_good')}
furniture=json.loads((root/'furniture.proposal.json').read_text())
proposed_furniture={g for r in furniture['recipes'] for g in r['outputs']}
# Only household-consumable products have retail channels. Bulk and intermediate
# products are B2B, even if they are terminal nodes in this limited catalog.
channels={
 'grocery_store':{'bread_pack','rye_bread','dry_pasta','barley_flour','oat_flakes','potato_starch','tomato_sauce','sunflower_oil_bottle','milk_powder'},
 'produce_store':{'carrot_pack','onion_pack','potato_pack','apple'},
 'dairy_store':{'milk_bottle','butter','cheese','sour_cream','yogurt','ice_cream','skimmed_milk'},
 'beverage_store':{'apple_juice_bottle','cocoa_drink_bottle'},
 'homeware_store':{'glassware','paper_bag','plastic_container'},
 'hardware_store':{'wood_pellets','lubricant_pack'},
 'building_materials_store':{'brick','ceramic_tile','concrete_block','mortar_mix','paving_slab','roof_tile','glass_window','plastic_pipe','plywood','particle_board'},
 'furniture_store':{'furniture_table','furniture_wardrobe','furniture_shelf','furniture_bed_frame','furniture_chair','furniture_desk','packed_furniture_table'},
 'agricultural_supply_store':{'animal_feed'},
}
# Optional retail availability of raw or intermediate goods requires a separate
# consumer packaging / safety approval; do not auto-classify them.
retail={g for goods in channels.values() for g in goods}
assert len(retail)==sum(map(len,channels.values())),'multiple primary retail channels'
assert retail <= produced|proposed_furniture
industrial=produced-retail
# Explicitly flag terminal goods without retail access; this is often correct B2B.
terminal=produced-consumed
terminal_b2b=sorted(terminal-retail)
store_defs={
 'grocery_store':'Продуктовый магазин', 'produce_store':'Овощной и фруктовый магазин',
 'dairy_store':'Молочный магазин', 'beverage_store':'Магазин напитков',
 'homeware_store':'Магазин товаров для дома', 'hardware_store':'Хозяйственный магазин',
 'building_materials_store':'Магазин строительных материалов',
 'furniture_store':'Мебельный магазин',
 'agricultural_supply_store':'Магазин товаров для сельского хозяйства'}
assert set(store_defs)==set(channels)
# Every output is accounted for: either explicit retail or B2B/wholesale.
assert produced == (produced&retail)|industrial
assert not (produced&retail)&industrial
entries=[]
for good in sorted(produced|proposed_furniture):
    stores=[s for s,goods in channels.items() if good in goods]
    entries.append({'good_id':good,'status':'proposed_not_active',
       'primary_channel':'retail' if stores else 'b2b_wholesale_contract_or_exchange',
       'retail_store_types':stores,
       'note':'Finished consumer good, conditional on packaging/safety' if stores else 'Industrial/intermediate or unclassified for consumer retail; may be sold B2B, not automatically in shops'})
result={'schema_version':1,'status':'design_proposal_not_active',
 'rules':['Store cannot create goods','Consumer demand only for retail-approved SKUs, not every recipe output',
 'Goods may have multiple approved store types later; primary assignment here is one per good',
 'B2B uses exchange/direct contracts, not household store demand',
 'Check stock, quality, safety, packaging, transport, city coverage, budgets and next-tick sales',
 'Retail classification does not activate any draft recipe'],
 'stores':[{'id':sid,'name_ru':store_defs[sid],'status':'proposed','goods':sorted(goods)} for sid,goods in channels.items()],
 'goods':entries,'terminal_goods_without_consumer_store':terminal_b2b,
 'requires_review':['Food safety and packaging for unwrapped or unpacked products',
 'Building materials shop versus project-based B2B procurement',
 'Animal feed packaging, intended buyer and safety','Household demand for fuel/industrial goods should not be assumed',
 'Multi-store cross-listing and city availability require explicit rules']}
(root/'retail-channels.proposal.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print(f'PASS: {len(produced)} catalog goods + {len(proposed_furniture)} furniture outputs; {len(channels)} store types; {len(retail)} retail SKUs; {len(terminal_b2b)} terminal B2B-only goods')
print('Terminal B2B:',', '.join(terminal_b2b))
