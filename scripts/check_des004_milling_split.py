#!/usr/bin/env python3
"""Check a proposed, non-executable co-product split for wheat milling."""
import json
from pathlib import Path

from generated_report import check_generated_report
root=Path(__file__).resolve().parents[1]/'docs/catalogs'
set_=json.loads((root/'game-production-profiles.v1.json').read_text())
p=next(p for p in set_['profiles'] if p['id']=='GAME-WHEAT-FLOUR-V1')
ref=next(p for p in json.loads((root/'pilot-evidence.json').read_text())['profiles'] if p['recipe_id']=='R-WHEAT-FLOUR')
assert p['batch_inputs']=={'wheat_grain':10000}
assert p['batch_outputs']=={'wheat_flour':7400}
assert p['unstocked_residue_g']==2600
fractions=ref['reference']['historical_fraction_split']['fraction_basis_points']
assert sum(fractions.values())==10000
# Model only a proposed recovery basis: keep pilot flour fixed, scale historical
# bran-to-flour ratio. Remaining recovered fractions are NOT silently saleable.
bran=round(p['batch_outputs']['wheat_flour']*fractions['bran']/fractions['flour'])
other=10000-7400-bran
assert 0<bran<2600 and other>=0
assert 7400+bran+other==10000
out={'status':'proposed_not_executable','recipe_id':'R-WHEAT-FLOUR','recipe_version':2,
 'basis':'game scenario scaled from historical recovered-product fractions, not a measured yield',
 'input_wheat_grain_g':10000,'flour_g':7400,'bran_g':bran,
 'other_unstocked_fractions_and_losses_g':other,
 'historical_fraction_source':ref['reference']['historical_fraction_split']['source'],
 'notes':'Do not add to approved production profiles until equipment applicability, recoverable bran quality, losses and market price are verified.'}
check_generated_report(root/'milling-split.proposal.json', json.dumps(out,ensure_ascii=False,indent=2)+'\n')
print('PASS: proposed mass balance',out['input_wheat_grain_g'],'=',out['flour_g'],'+',out['bran_g'],'+',out['other_unstocked_fractions_and_losses_g'])
