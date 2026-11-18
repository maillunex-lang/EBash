#!/usr/bin/env python3
"""Draft furniture factory + retail design, independent of executable recipe catalog."""
import json
from pathlib import Path

from generated_report import check_generated_report
root=Path(__file__).resolve().parents[1]/'docs/catalogs'
base=json.loads((root/'recipes.seed.json').read_text())['recipes']
existing={r['id'] for r in base}
resources={r['id'] for r in json.loads((root/'resources.seed.json').read_text())['resources']}
# Explicit counts for finished goods, grams for input stock. Assembly does not
# claim mass conservation against pieces, nor convert site/utility to stock.
recipes=[
 ('R-FURN-PANEL-CUT',{'particle_board':20000},{'furniture_panel_set':1},'panel_saw',420),
 ('R-FURN-TABLE',{'furniture_panel_set':1,'furniture_fittings':1,'surface_finish':250},{'furniture_table':1},'assembly_bench',260),
 ('R-FURN-WARDROBE',{'particle_board':45000,'furniture_fittings':2,'surface_finish':500},{'furniture_wardrobe':1},'panel_saw_assembly',850),
 ('R-FURN-SHELF',{'sawn_timber':12000,'furniture_fittings':1,'surface_finish':200},{'furniture_shelf':1},'woodshop',330),
 ('R-FURN-BED',{'sawn_timber':35000,'furniture_fittings':2,'surface_finish':350},{'furniture_bed_frame':1},'woodshop',730),
 ('R-FURN-CHAIR',{'sawn_timber':8000,'furniture_fittings':1,'surface_finish':150},{'furniture_chair':1},'woodshop',260),
 ('R-FURN-DESK',{'plywood':16000,'furniture_fittings':1,'surface_finish':220},{'furniture_desk':1},'woodshop',420),
 ('R-FURN-PACK',{'furniture_table':1,'carton_box':1},{'packed_furniture_table':1},'packing_bench',45),
]
entries=[]
newgoods={'furniture_panel_set','furniture_fittings','surface_finish','furniture_table','furniture_wardrobe','furniture_shelf','furniture_bed_frame','furniture_chair','furniture_desk','packed_furniture_table'}
for rid,inputs,outputs,equipment,energy in recipes:
    assert rid not in existing
    assert all(isinstance(n,int) and n>0 for n in [*inputs.values(),*outputs.values()])
    assert all(k in resources or k in newgoods for k in inputs)
    assert all(k in newgoods for k in outputs)
    entries.append({'id':rid,'version':1,'family':'furniture','status':'draft_not_executable',
      'inputs':inputs,'outputs':outputs,'equipment_class_hypothesis':equipment,
      'energy_wh_per_batch_hypothesis':energy,'workers_per_line_hypothesis':2,
      'minimum_batches_hypothesis':1,'quality_profile':None,'safety_profile':None,
      'unit_note':'wood panels/finish are grams; fittings, kits, finished furniture and boxes are counts',
      'source_status':'game_design_hypothesis_requires_verification'})
assert len(entries)==8
# Check dependencies: factory production must not silently invent materials.
producers={g:r['id'] for r in entries for g in r['outputs']}
external=sorted({g for r in entries for g in r['inputs'] if g not in resources and g not in producers})
assert external==['furniture_fittings','surface_finish']
for r in entries:
    for g in r['inputs']:
        if g in producers:assert producers[g]!=r['id']
# Explicit store: transfers, stocks, next-tick sales; no store recipes.
store={'id':'FAC-FURNITURE-STORE','kind':'retail_store','status':'proposed_not_executable',
 'sells':['furniture_table','furniture_wardrobe','furniture_shelf','furniture_bed_frame','furniture_chair','furniture_desk','packed_furniture_table'],
 'receives':'confirmed deliveries of finished furniture from a factory or wholesale warehouse',
 'constraints':['no goods created by store','sale only after next tick for produced goods',
 'atomic reservation and sale of stock','city coverage and household budgets',
 'goods compatibility, safety and quality','transport, stock capacity and shop operating costs']}
result={'schema_version':1,'status':'design_proposal_not_executable','facility':{'id':'FAC-FURNITURE-FACTORY','kind':'factory','layout':'3x3 production stages subject to equipment compatibility and capacity','status':'proposed'},
 'recipes':entries,'new_resource_candidates':sorted(newgoods),'external_goods_without_supply_recipes':external,
 'retail_facility':store,'activation_blockers':['source and verify material quantities and yields',
 'define fittings/finish suppliers and units','test 3x3 line placement, equipment, energy, workers',
 'define waste, quality, safety and durability','price/cost balance and population demand',
 'implement production and store inventory integration before enabling']}
check_generated_report(root/'furniture.proposal.json', json.dumps(result,ensure_ascii=False,indent=2)+'\n')
print('PASS: 8 furniture drafts, factory, retail store, 10 resource candidates; external supply gaps:',', '.join(external))
