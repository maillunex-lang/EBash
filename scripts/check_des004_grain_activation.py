#!/usr/bin/env python3
"""Check approved bread profiles against draft grain recipes without activation."""
import json
from pathlib import Path
root = Path(__file__).resolve().parents[1] / 'docs/catalogs'
drafts = {r['id']: r for r in json.loads((root/'recipes.seed.json').read_text())['recipes'] if r['family']=='grain'}
profiles = json.loads((root/'game-production-profiles.v1.json').read_text())['profiles']
evidence = {(p['recipe_id'],p['recipe_version']):p for p in json.loads((root/'pilot-evidence.json').read_text())['profiles']}
assert len(drafts)==16
report=['# DES-004: совместимость игровых профилей зерновой отрасли','','Это предактивационная проверка, а не разрешение исполнять черновики.','']
blockers=0
for p in profiles:
    recipe=drafts.get(p['recipe_id'])
    if recipe is None: continue
    issues=[]
    if p['recipe_version']!=recipe['version']: issues.append('версия не совпадает')
    inputs={i['good_id'] for i in recipe['inputs'] if i['input_kind'] not in ('site','utility')}
    outputs={i['good_id'] for i in recipe['outputs']}
    if set(p['batch_inputs'])!=inputs: issues.append('расхождение входов')
    if set(p['batch_outputs'])!=outputs: issues.append('расхождение выходов: '+str(sorted(outputs-set(p['batch_outputs']))))
    for i in recipe['inputs']:
        if i['input_kind'] not in ('site','utility') and i['quantity'] is not None and i['unit']=='g' and p['batch_inputs'].get(i['good_id'])!=i['quantity']:
            issues.append('расхождение количества '+i['good_id'])
    ref=evidence[(p['recipe_id'],p['recipe_version'])]
    masses=ref['reference'].get('ingredient_mass_g')
    if masses and masses!=p['batch_inputs']: issues.append('расхождение со справочной рецептурой')
    if issues: blockers+=1
    report.append('- '+p['recipe_id']+' v'+str(p['recipe_version'])+': '+ ('БЛОКЕР: '+', '.join(issues) if issues else 'согласованы проверяемые поля'))
report+=['','Несогласованных профилей: '+str(blockers)+'.','Совместные выходы требуют явного материального баланса. Черновики не активированы.','']
(root/'grain-activation-gate.md').write_text('\n'.join(report),encoding='utf-8')
print(f'PASS: {len(profiles)} profiles, {len(drafts)} grain drafts; blockers={blockers}')
