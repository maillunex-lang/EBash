#!/usr/bin/env python3
"""Validate proposed game-only production profiles without unlocking DES-004."""
import json
from pathlib import Path
from collections import Counter
root=Path(__file__).resolve().parents[1]/'docs'/'catalogs'
data=json.loads((root/'game-production-profiles.v1.json').read_text())
recipes={r['id']:r for r in json.loads((root/'recipes.seed.json').read_text())['recipes']}
assert data['status']=='approved_game_balance_v1'
assert sum(data['production_rules']['quality_weights_percent'].values())==100
profiles=data['profiles']; assert len({p['id'] for p in profiles})==len(profiles)
for p in profiles:
 r=recipes[p['recipe_id']]
 assert r['version']==p['recipe_version'] and r['status']=='draft_not_executable'
 assert p['minimum_batches']>=1 and p['equipment_capacity_batches_per_tick']>=p['minimum_batches']
 assert p['energy_wh_per_batch']>0 and p['workers_per_active_line']>=1
 assert 1<=p['quality_floor']<=p['quality_reference']<=100
 assert p['game_operation_cost_kopecks']>=0
 assert all(isinstance(n,int) and n>0 for n in (*p['batch_inputs'].values(),*p['batch_outputs'].values()))
 assert set(p['batch_inputs'])=={x['good_id'] for x in r['inputs'] if x['input_kind']!='utility'}
 assert set(p['batch_outputs'])<={x['good_id'] for x in r['outputs']}
 output_mass=sum(p['batch_outputs'].values()) if 'assumed_loaf_mass_g' not in p else p['batch_outputs']['bread_loaf']*p['assumed_loaf_mass_g']
 assert sum(p['batch_inputs'].values())==output_mass+p.get('unstocked_residue_g',0)+p.get('unstocked_bake_loss_g',0),p['id']
stock=Counter({'wheat_grain':10000,'water':5440,'salt':144,'yeast':112})
for p,n in zip(profiles,(1,8,8)):
 assert n<=p['equipment_capacity_batches_per_tick']
 for k,v in p['batch_inputs'].items():
  assert stock[k]>=v*n
  stock[k]-=v*n
 for k,v in p['batch_outputs'].items():stock[k]+=v*n
assert +stock==Counter({'wheat_flour':200,'bread_loaf':24}),stock
assert sum(p['game_operation_cost_kopecks']*n for p,n in zip(profiles,(0,8,8)))==14000
assert sum(p['energy_wh_per_batch']*n for p,n in zip(profiles,(1,8,8)))==4260
ref=data['economy_reference']
assert ref['starting_cash_kopecks']-ref['grain_purchase_kopecks_per_10000g']-ref['other_ingredients_purchase_kopecks_for_8_portions']-ref['flour_delivery_kopecks']-ref['total_mixing_and_baking_kopecks_for_8_portions']+24*ref['loaf_sale_price_kopecks']==ref['reference_cash_after_sale_kopecks']
print('PASS: game profiles mass, equipment, energy, workers, money and DES-004 draft guard')
