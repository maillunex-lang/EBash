#!/usr/bin/env python3
"""Validate 16 explicitly hypothetical, non-executable grain production scenarios."""
import json
from collections import defaultdict
from pathlib import Path
root=Path(__file__).resolve().parents[1]/'docs/catalogs'
recipes={r['id']:r for r in json.loads((root/'recipes.seed.json').read_text())['recipes'] if r['family']=='grain'}
assert len(recipes)==16
# Units: grams per game batch; loaves/packs are count units.
# Field harvest is a hypothetical site yield, NOT consumption of the site.
spec={
 'R-WHEAT-GRAIN':({'wheat_field':1},{'wheat_grain':10000},'field',250,2,1),
 'R-WHEAT-FLOUR':({'wheat_grain':10000},{'wheat_flour':7400,'wheat_bran':1333},'mill',500,1,1),
 'R-BREAD-DOUGH':({'wheat_flour':900,'water':680,'yeast':14,'salt':18},{'bread_dough':1612},'mixer',32,1,1),
 'R-BREAD-LOAF':({'bread_dough':1612},{'bread_loaf':3},'oven',420,1,1),
 'R-BREAD-PACK':({'bread_loaf':3,'paper_pack':3},{'bread_pack':3},'packing',20,1,1),
 'R-BARLEY-GRAIN':({'barley_field':1},{'barley_grain':10000},'field',250,2,1),
 'R-BARLEY-FLOUR':({'barley_grain':10000},{'barley_flour':7600},'mill',500,1,1),
 'R-RYE-GRAIN':({'rye_field':1},{'rye_grain':10000},'field',250,2,1),
 'R-RYE-FLOUR':({'rye_grain':10000},{'rye_flour':7500},'mill',500,1,1),
 'R-RYE-DOUGH':({'rye_flour':900,'water':720,'yeast':14},{'rye_dough':1634},'mixer',35,1,1),
 'R-RYE-BREAD':({'rye_dough':1634},{'rye_bread':3},'oven',450,1,1),
 'R-OAT-GRAIN':({'oat_field':1},{'oat_grain':10000},'field',250,2,1),
 'R-OAT-FLAKES':({'oat_grain':10000},{'oat_flakes':7600},'roller',550,1,1),
 'R-ANIMAL-FEED':({'wheat_bran':400,'barley_grain':600},{'animal_feed':1000},'feed_mixer',80,1,1),
 'R-PASTA-DOUGH':({'wheat_flour':1000,'water':450},{'pasta_dough':1450},'pasta_mixer',70,1,1),
 'R-DRY-PASTA':({'pasta_dough':1450},{'dry_pasta':1120},'dryer',800,1,1),
}
assert set(spec)==set(recipes)
mass_units={'wheat_grain','wheat_flour','wheat_bran','water','yeast','salt','bread_dough','barley_grain','barley_flour','rye_grain','rye_flour','rye_dough','oat_grain','oat_flakes','animal_feed','pasta_dough','dry_pasta'}
count_units={'bread_loaf','paper_pack','bread_pack'}
out=[]
for rid,(inputs,outputs,equip,energy,workers,min_batch) in spec.items():
    draft=recipes[rid]
    declared_in={i['good_id'] for i in draft['inputs'] if i['input_kind']!='utility'}
    declared_out={i['good_id'] for i in draft['outputs']}
    assert set(inputs)==declared_in,(rid,'inputs')
    assert set(outputs)==declared_out,(rid,'outputs')
    assert energy>0 and workers>0 and min_batch>0
    for good,qty in {**inputs,**outputs}.items():
        assert isinstance(qty,int) and qty>0,(rid,good)
    site={i['good_id'] for i in draft['inputs'] if i['input_kind']=='site'}
    utility={i['good_id'] for i in draft['inputs'] if i['input_kind']=='utility'}
    # Utility is a service; no inventory item should be emitted.
    assert not site.intersection(outputs) and not utility.intersection(outputs)
    # Do not treat bread/pack counts as grams; mass check is only on comparable mass goods.
    mass_in=sum(v for k,v in inputs.items() if k in mass_units)
    mass_out=sum(v for k,v in outputs.items() if k in mass_units)
    if not site and not count_units.intersection(set(inputs)|set(outputs)) and mass_in:
        assert mass_out<=mass_in,(rid,mass_in,mass_out)
    residue=mass_in-mass_out if mass_in and not count_units.intersection(set(inputs)|set(outputs)) else None
    out.append({'recipe_id':rid,'recipe_version':draft['version'],'status':'proposed_not_executable',
      'batch_inputs':{k:v for k,v in inputs.items() if k not in site},
      'site_requirement':{k:inputs[k] for k in site},
      'utility_requirements':sorted(utility),'batch_outputs':outputs,
      'equipment_class_hypothesis':equip,'energy_wh_per_batch_hypothesis':energy,
      'workers_per_line_hypothesis':workers,'minimum_batches_hypothesis':min_batch,
      'quality_rule':'unverified; must be explicitly specified before activation',
      'mass_residue_g_if_comparable':residue,
      'source_status':'game_design_hypothesis_not_industrial_measurement'})
# Run a bounded chain simulation with existing material from the separate milling proposal.
mill=next(x for x in out if x['recipe_id']=='R-WHEAT-FLOUR')
assert mill['batch_outputs']=={'wheat_flour':7400,'wheat_bran':1333}
feed=next(x for x in out if x['recipe_id']=='R-ANIMAL-FEED')
assert feed['batch_inputs']=={'wheat_bran':400,'barley_grain':600}
assert feed['batch_outputs']['animal_feed']==1000
pasta=next(x for x in out if x['recipe_id']=='R-PASTA-DOUGH')
dry=next(x for x in out if x['recipe_id']=='R-DRY-PASTA')
assert pasta['batch_outputs']['pasta_dough']==dry['batch_inputs']['pasta_dough']
assert pasta['batch_inputs']['wheat_flour']<=mill['batch_outputs']['wheat_flour']
assert dry['batch_outputs']['dry_pasta']<=dry['batch_inputs']['pasta_dough']
result={'status':'proposed_not_executable','scope':'16 grain scenarios, not approved executable recipes',
 'units':'grams for mass goods; counts for loaves/packs; site requirement is a location/capacity right',
 'assumptions':'All quantities except explicitly sourced bread/milling pilot values are provisional game balance hypotheses; no safety or equipment compatibility certification.',
 'profiles':out,'activation_blockers':['quality and food/feed safety','site/crop yield rules','equipment compatibility and 3x3 layout','loss/residue classification','economic pricing and margin sensitivity','warehouse lots, reservations, sale delay, delivery integration']}
(root/'grain-scenarios.proposal.json').write_text(json.dumps(result,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print('PASS: 16/16 grain proposals, graph inputs/outputs, comparable mass conservation, feed and pasta chain')
