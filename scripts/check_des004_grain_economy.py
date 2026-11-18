#!/usr/bin/env python3
"""Sensitivity and resource stress for non-executable DES-004 grain scenarios."""
import json
from collections import Counter
from fractions import Fraction
from pathlib import Path
root=Path(__file__).resolve().parents[1]/'docs/catalogs'
proposed=json.loads((root/'grain-scenarios.proposal.json').read_text())
assert proposed['status']=='proposed_not_executable'
profiles={p['recipe_id']:p for p in proposed['profiles']}
assert len(profiles)==16
# All prices are hypothetical kopecks/kg, not actual game market prices.
prices={'wheat_grain':1500,'barley_grain':1000,'rye_grain':1400,'oat_grain':1000,
 'wheat_flour':2400,'wheat_bran':300,'barley_flour':1800,'rye_flour':2400,
 'oat_flakes':2400,'animal_feed':1000,'dry_pasta':3400,
 'water':10,'yeast':20000,'salt':200,'pasta_dough':0,'rye_dough':0,'bread_dough':0}
# Operational costs intentionally zero in the optimistic bound; if even this
# loses money, real costs cannot rescue the chain at unchanged prices.
chains=[('wheat milling',['R-WHEAT-FLOUR'],{'wheat_grain':10000}),
 ('barley milling',['R-BARLEY-FLOUR'],{'barley_grain':10000}),
 ('rye milling',['R-RYE-FLOUR'],{'rye_grain':10000}),
 ('oat flakes',['R-OAT-FLAKES'],{'oat_grain':10000}),
 ('dry pasta',['R-PASTA-DOUGH','R-DRY-PASTA'],{'wheat_flour':1000,'water':450}),
 ('feed',['R-ANIMAL-FEED'],{'wheat_bran':400,'barley_grain':600})]
results=[]
for name,stages,initial in chains:
    stock=Counter(initial)
    for rid in stages:
        p=profiles[rid]
        before=stock.copy()
        for k,n in p['batch_inputs'].items():
            assert stock[k]>=n,(name,rid,k)
        for k,n in p['batch_inputs'].items():stock[k]-=n
        for k,n in p['batch_outputs'].items():stock[k]+=n
        assert all(n>=0 for n in stock.values()),rid
        # Same tick reexecution is disallowed; only one execution per stage here.
    revenue=sum(Fraction(n*prices[k],1000) for k,n in stock.items() if n>0 and k in prices)
    purchased=sum(Fraction(n*prices[k],1000) for k,n in initial.items())
    optimistic_margin=revenue-purchased
    results.append({'chain':name,'input_stock_g':initial,'ending_stock':{k:v for k,v in stock.items() if v>0},
                    'gross_margin_before_energy_labor_capex_and_tax_kopecks':str(optimistic_margin),
                    'negative_even_without_operating_costs':optimistic_margin<0})
assert len(results)==6
assert all('gross_margin_before_energy_labor_capex_and_tax_kopecks' in x for x in results)
# Shared batch constraints: no materials and no energy means no execution.
for p in profiles.values():
    assert p['energy_wh_per_batch_hypothesis']>0
    assert p['workers_per_line_hypothesis']>0
    assert p['minimum_batches_hypothesis']>=1
    assert p['equipment_class_hypothesis']
# Land harvest requires a site; it must not mint grain without a real land right.
for rid in ('R-WHEAT-GRAIN','R-BARLEY-GRAIN','R-RYE-GRAIN','R-OAT-GRAIN'):
    assert profiles[rid]['site_requirement']
    assert not profiles[rid]['batch_inputs']
report={'status':'analysis_only','pricing':'illustrative_kopecks_per_kg_not_market_data',
 'assumption':'All scenarios ignore energy, workers, capital costs and taxes: gross margin is an optimistic upper bound, not profit.',
 'price_assumptions':prices,'chains':results,
 'blocking_findings':['Gross margins must be stress-tested against operating costs and price changes',
 'Site-based grain generation is not integrated with land ownership and crop time',
 'Bread count units and paper packaging need a separate price and stock model',
 'Missing energy tariff, labor, capital, storage and logistics parameters',
 'Quality and feed safety not certified',
 'Do not execute proposed recipes until atomic inventory and next-tick sale tests pass']}
(root/'grain-economy.audit.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
for r in results:print('CHECK:',r['chain'],r['gross_margin_before_energy_labor_capex_and_tax_kopecks'],'kopecks optimistic margin')
print('PASS: 6 scenarios, nonnegative stock, 16 equipment/energy/staff constraints, crop site requirements')
