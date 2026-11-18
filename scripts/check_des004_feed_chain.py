#!/usr/bin/env python3
"""Validate a NON-EXECUTABLE game-only bran/barley/feed scenario and conservation."""
import json
from pathlib import Path
from fractions import Fraction
root=Path(__file__).resolve().parents[1]/'docs/catalogs'
recipes={r['id']:r for r in json.loads((root/'recipes.seed.json').read_text())['recipes']}
milling=json.loads((root/'milling-split.proposal.json').read_text())
assert milling['status']=='proposed_not_executable'
assert milling['input_wheat_grain_g']==milling['flour_g']+milling['bran_g']+milling['other_unstocked_fractions_and_losses_g']
assert {i['good_id'] for i in recipes['R-ANIMAL-FEED']['inputs']}=={'wheat_bran','barley_grain'}
assert {o['good_id'] for o in recipes['R-ANIMAL-FEED']['outputs']}=={'animal_feed'}
# Explicit game hypotheses, not feed standards. 1000 g dry feed per batch.
feed={'wheat_bran_g':400,'barley_grain_g':600,'animal_feed_g':1000,'energy_wh':80,'operators':1,'min_batches':1,'equipment':'feed_mixer'}
assert feed['wheat_bran_g']+feed['barley_grain_g']==feed['animal_feed_g']
# Grain from field is deliberately external to this test: its production norms are missing.
bran=milling['bran_g']; barley=1200; produced=0
while bran>=feed['wheat_bran_g'] and barley>=feed['barley_grain_g']:
    bran-=feed['wheat_bran_g'];barley-=feed['barley_grain_g'];produced+=feed['animal_feed_g']
assert (produced,bran,barley)==(2000,533,0)
assert produced+bran+barley==milling['bran_g']+1200
# Avoid double-crediting by-products: total flour + bran + other equals input grain.
# Marginal contribution of feed must pay for purchased barley and processing.
# All values below are *illustrative kopecks*, NOT game market prices.
prices={'wheat_grain_kopecks_per_kg':1500,'flour_kopecks_per_kg':2400,
        'bran_kopecks_per_kg':300,'barley_kopecks_per_kg':1000,
        'feed_kopecks_per_kg':1000,'milling_cost_kopecks':12000,
        'feed_cost_kopecks_per_batch':200}
value=lambda grams,rate:Fraction(grams*rate,1000)
base=value(milling['flour_g'],prices['flour_kopecks_per_kg'])-value(10000,prices['wheat_grain_kopecks_per_kg'])-prices['milling_cost_kopecks']
bran_sale=base+value(milling['bran_g'],prices['bran_kopecks_per_kg'])
feed_case=base+value(produced,prices['feed_kopecks_per_kg'])+value(bran,prices['bran_kopecks_per_kg'])-value(1200,prices['barley_kopecks_per_kg'])-2*prices['feed_cost_kopecks_per_batch']
assert feed_case-bran_sale==value(produced,prices['feed_kopecks_per_kg'])-value(800,prices['bran_kopecks_per_kg'])-value(1200,prices['barley_kopecks_per_kg'])-400
assert feed_case>bran_sale # positive incremental margin here; requires sensitivity tests
report={'status':'proposed_not_executable','recipe_id':'R-ANIMAL-FEED','recipe_version':recipes['R-ANIMAL-FEED']['version'],
 'game_only_hypothesis':feed,'scenario':{'wheat_grain_input_g':10000,'flour_output_g':7400,'bran_output_g':1333,'unstocked_milling_fractions_g':1267,'barley_grain_purchased_g':1200,'feed_output_g':produced,'bran_remaining_g':bran},
 'illustrative_prices_not_approved':prices,'profit_kopecks':{'milling_without_bran':str(base),'milling_plus_bran_sale':str(bran_sale),'milling_plus_feed':str(feed_case)},
 'required_before_activation':['validate feed safety and nutritional composition','validate production equipment and energy','approve game price policy and ensure co-products not double sold','define barley field production separately','test inventory lots, reservations and next-tick sales']}
(root/'feed-chain.proposal.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
print('PASS: 10kg wheat -> 7.4kg flour + 1.333kg bran + 1.267kg other; 1.2kg purchased barley + 0.8kg bran -> 2kg feed; positive illustrative incremental feed margin (not a market forecast)')
