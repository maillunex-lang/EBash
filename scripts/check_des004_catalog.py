#!/usr/bin/env python3
"""Validate draft catalog identifiers, dependency graph and fail-closed readiness."""
import argparse
import json
from collections import Counter, defaultdict
from fractions import Fraction
from pathlib import Path
from urllib.parse import urlsplit


def validate(*, details=False, chain=None, batch=False):
    root=Path(__file__).resolve().parent.parent/'docs'/'catalogs'
    data=json.loads((root/'recipes.seed.json').read_text())
    resources=json.loads((root/'resources.seed.json').read_text())['resources']
    recipes=data['recipes']
    ids=[r['id'] for r in recipes]
    if chain is not None and chain not in ids:
        raise SystemExit(f'Unknown recipe ID: {chain}')
    assert len(ids)==len(set(ids)) and len(ids)>=100
    assert all(type(r['version']) is int and r['version'] > 0 and r['status']=='draft_not_executable' for r in recipes)
    assert all(r['inputs'] and r['outputs'] and r['source_status']=='requires_verification' for r in recipes)
    profile_fields = (
        'min_batch', 'equipment_profile', 'energy_profile',
        'workforce_profile', 'quality_profile',
    )
    missing = Counter()
    blockers = {}
    for recipe in recipes:
        unresolved = [field for field in profile_fields if recipe[field] is None]
        missing.update(unresolved)
        blockers[recipe['id']] = unresolved
        assert all(key in item for item in recipe['inputs'] + recipe['outputs']
                   for key in ('quantity', 'unit')), recipe['id']
        goods_in = [item for item in recipe['inputs']
                 if item['input_kind'] in ('produced_good', 'external_good')]
        goods = goods_in + recipe['outputs']
        if any(item['quantity'] is None or item['unit'] is None for item in goods):
            missing['material_quantity_or_unit'] += 1
            unresolved.extend(
                f"{kind} {item['good_id']}: "
                + ', '.join(field for field in ('quantity', 'unit') if item[field] is None)
                for kind, items in (('input', goods_in),
                                    ('output', recipe['outputs']))
                for item in items if item['quantity'] is None or item['unit'] is None
            )
        assert any(recipe[field] is None for field in profile_fields), (
            f"{recipe['id']}: complete profiles in the draft catalog require review"
        )
    evidence = json.loads((root / 'pilot-evidence.json').read_text())
    assert evidence['schema_version'] == 1 and evidence['status'] == 'reference_only'
    versions = {(r['id'], r['version']): r for r in recipes}
    keys = set()
    for entry in evidence['profiles']:
        key = (entry['recipe_id'], entry['recipe_version'])
        assert key in versions and key not in keys, key
        keys.add(key)
        assert entry['source'].startswith('https://') and entry['unverified'], key
        masses = entry['reference'].get('ingredient_mass_g')
        if masses is not None:
            assert set(masses) == {i['good_id'] for i in versions[key]['inputs']}, key
            assert all(type(value) is int and value > 0 for value in masses.values()), key
            for ingredient in versions[key]['inputs']:
                if ingredient['quantity'] is not None:
                    assert ingredient['unit'] == 'g', key
                    assert ingredient['quantity'] == masses[ingredient['good_id']], key
        equipment = entry.get('equipment_reference')
        if equipment is not None:
            assert masses and 'wheat_flour' in masses, key
            portions = equipment['control_recipe_portions']
            assert type(portions) is int and portions > 0, key
            assert portions * sum(masses.values()) <= equipment['max_dough_g'], key
            assert portions * masses['wheat_flour'] <= equipment['max_flour_g'], key
            limits = (equipment['max_dough_g'] // sum(masses.values()),
                      equipment['max_flour_g'] // masses['wheat_flour'])
            print(f"PASS: {equipment['model']} control batch of {portions} portions; "
                  f"catalog capacity bound {min(limits)} portions")
    assert keys == versions.keys(), f'Missing reference profiles: {sorted(versions.keys() - keys)}'
    print(f'PASS: all {len(keys)} draft recipe versions have reference profiles')
    # Replay the documented pilot handoff from milling to mixing.
    pilot = {entry['recipe_id']: entry for entry in evidence['profiles']}
    drying = pilot['R-DRY-PASTA']['reference']['drying_moisture']
    initial_moisture = drying['approximate_input_percent']
    low, high = drying['output_percent_range']
    assert 0 <= low <= high < initial_moisture < 100
    dry_matter_g = 100_000 * (100 - initial_moisture) // 100
    output_range_g = tuple(dry_matter_g * 100 // (100 - moisture)
                           for moisture in (low, high))
    assert output_range_g == (78_409, 79_310)
    tomato = pilot['R-TOMATO-PASTE']['reference']
    low_ratio, high_ratio = tomato['processing_ratio']['tomato_kg_per_paste_kg_range']
    low_screenings, high_screenings = tomato['processing_ratio']['screenings_percent_range']
    assert 0 < low_ratio <= high_ratio and 0 <= low_screenings <= high_screenings <= 100
    tomato_input_g = 100_000
    paste_range_g = (Fraction(tomato_input_g, high_ratio), Fraction(tomato_input_g, low_ratio))
    screenings_range_g = tuple(tomato_input_g * percent // 100
                               for percent in (low_screenings, high_screenings))
    assert paste_range_g == (Fraction(100_000, 7), 20_000)
    assert screenings_range_g == (3_000, 4_000)
    assert paste_range_g[1] + screenings_range_g[1] <= tomato_input_g
    solids = tomato['soluble_solids']
    assert solids['minimum_natural_soluble_solids_percent'] == 24
    print('PASS: reference pasta moisture balance and tomato paste yield range and solids threshold')
    paper = pilot['R-PAPER']['reference']
    wet_web_kg = 100
    dry_solids_kg = Fraction(wet_web_kg * paper['approximate_inlet_dry_solids_percent'], 100)
    pressed_web_kg = dry_solids_kg * 100 / paper['approximate_outlet_dry_solids_percent']
    assert (pressed_web_kg, wet_web_kg - pressed_web_kg) == (40, 60)
    butter = pilot['R-BUTTER']['reference']
    cream_fat_bp = pilot['R-CREAM']['reference']['fat_basis_points']['cream']
    upper_butter_kg = Fraction(cream_fat_bp, butter['minimum_milkfat_percent'])
    assert upper_butter_kg == 50
    fat_loss = butter['churning_example']['cream_fat_to_buttermilk_basis_points']
    assert upper_butter_kg * Fraction(10_000 - fat_loss, 10_000) == Fraction(199, 4)
    assert {item['good_id'] for item in versions[('R-BUTTER', pilot['R-BUTTER']['recipe_version'])]['outputs']} == {'butter', 'buttermilk'}
    cement = versions[('R-CEMENT', pilot['R-CEMENT']['recipe_version'])]
    assert any(item['good_id'] == 'gypsum' and item['input_kind'] == 'external_good'
               for item in cement['inputs'])
    benchmark = pilot['R-REFINED-FUEL']['reference']['refinery_volume_benchmark']
    assert benchmark['approximate_total_product_gallons'] - benchmark['crude_gallons'] == 3
    print('PASS: wet paper pressing, butter mass bound and aggregate refinery volume')
    milling = pilot['R-WHEAT-FLOUR']['reference']
    split = milling['historical_fraction_split']
    assert split['basis'] == 'total_recovered_products'
    fractions = split['fraction_basis_points']
    assert set(fractions) == {'flour', 'bran', 'shorts', 'red_dog', 'germ'}
    assert all(type(value) is int and value > 0 for value in fractions.values())
    assert sum(fractions.values()) == 10_000
    print('PASS: measured milling fractions reconcile on recovered-product basis')
    extraction = milling['laboratory_extraction_percent']
    dough = pilot['R-BREAD-DOUGH']
    ingredients = dough['reference']['ingredient_mass_g']
    equipment = dough['equipment_reference']
    grain_g = 10_000
    flour_g = grain_g * extraction // 100
    portions = min(flour_g // ingredients['wheat_flour'],
                   equipment['max_flour_g'] // ingredients['wheat_flour'],
                   equipment['max_dough_g'] // sum(ingredients.values()))
    assert portions == equipment['control_recipe_portions'] == 8
    assert flour_g - portions * ingredients['wheat_flour'] == 200
    assert grain_g - flour_g == 2_600
    assert portions * sum(ingredients.values()) == 12_896
    flour_left_g = flour_g - portions * ingredients['wheat_flour']
    dough_g = portions * sum(ingredients.values())
    loaves = portions * 3
    baking = pilot['R-BREAD-LOAF']
    oven = baking['oven_reference']
    width, depth = oven['tray_mm']
    slot_width, slot_depth = baking['layout_assumption']['slot_mm']
    loaves_per_cycle = oven['trays'] * (width // slot_width) * (depth // slot_depth)
    full_cycles, last_load = divmod(loaves, loaves_per_cycle)
    assert (loaves_per_cycle, full_cycles, last_load) == (16, 1, 8)
    print('PASS: provisional oven layout, two loads of 16 and 8')
    bread_g = loaves * 475
    bake_loss_g = dough_g - bread_g
    other_ingredients_g = portions * (sum(ingredients.values()) - ingredients['wheat_flour'])
    assert bake_loss_g == 1_496
    assert grain_g + other_ingredients_g == bread_g + bake_loss_g + flour_left_g + grain_g - flour_g
    reference_min_loss_percent = pilot['R-BREAD-LOAF']['reference']['example_minimum_bake_weight_loss_percent']
    assert dough_g * (100 - reference_min_loss_percent) // 100 == 10_961

    grain_cost, ingredients_cost, delivery_cost, line_cost = 30_000, 18_000, 12_000, 14_000
    costs = grain_cost + ingredients_cost + delivery_cost + line_cost
    remaining_value = round(Fraction((grain_cost + delivery_cost) * flour_left_g, flour_g))
    revenue = loaves * 4_500
    assert (revenue, remaining_value, revenue - costs + remaining_value) == (108_000, 1_135, 35_135)
    assert 200_000 - costs + revenue == 234_000
    print('PASS: bread mass, stock valuation, revenue and closing cash')

    fat = pilot['R-CREAM']['reference']['fat_basis_points']
    raw_g = 79_900
    cream_g, remainder = divmod(raw_g * (fat['raw_milk'] - fat['skimmed_milk']),
                                fat['cream'] - fat['skimmed_milk'])
    assert remainder == 0
    skim_g = raw_g - cream_g
    assert (cream_g, skim_g) == (7_900, 72_000)
    mixed_cream_g, mixed_skim_g = 2_950, 37_000
    mixed_g = mixed_cream_g + mixed_skim_g
    cream_left_g, skim_left_g = cream_g - mixed_cream_g, skim_g - mixed_skim_g
    assert (cream_left_g, skim_left_g) == (4_950, 35_000)
    assert mixed_g + cream_left_g + skim_left_g == raw_g
    assert (mixed_cream_g * fat['cream'] + mixed_skim_g * fat['skimmed_milk']
            == mixed_g * 300)
    assert (mixed_g * 300 + cream_left_g * fat['cream'] + skim_left_g * fat['skimmed_milk']
            == raw_g * fat['raw_milk'])
    print('PASS: separation and standardization, mass and fat conserved')

    cell_counts = Counter()
    cells_by_recipe = {}
    for recipe in recipes:
        goods_in = {item['good_id'] for item in recipe['inputs']
                    if item['input_kind'] in ('external_good', 'produced_good')}
        goods_out = {item['good_id'] for item in recipe['outputs']}
        # Isolated-stage layout: one production cell and separate receiving
        # and shipping cells per material. Equipment compatibility is separate.
        slots = len(goods_in) + 1 + len(goods_out)
        assert slots <= 9, f"{recipe['id']}: {slots} cells exceed one 3×3 grid"
        cell_counts[slots] += 1
        cells_by_recipe[recipe['id']] = slots
    print('Draft 3×3 layouts by occupied cells:', dict(sorted(cell_counts.items())))
    output_to_recipe={}
    for r in recipes:
        for o in r['outputs']:
            assert o['good_id'] not in output_to_recipe, 'multiple producers need explicit modeling'
            output_to_recipe[o['good_id']]=r['id']
    registry={r['id']:r for r in resources}
    assert len(registry)==len(resources)
    used={x['good_id'] for r in recipes for x in r['inputs']+r['outputs']}
    assert used==set(registry), 'registry and recipe graph differ'
    kinds={'site','utility','external_good','produced_good'}
    assert all(r['kind'] in kinds and r['unit'] is None for r in resources)
    graph={r['id']:set() for r in recipes}
    for r in recipes:
        for x in r['inputs']:
            rid=x['good_id']
            assert x['input_kind']==registry[rid]['kind']
            if x['input_kind']=='produced_good':
                assert rid in output_to_recipe
                graph[r['id']].add(output_to_recipe[rid])
        for x in r['outputs']:
            assert registry[x['good_id']]['kind']=='produced_good'
    visited = set()
    visiting = set()
    ordered = []
    def dfs(rid):
        if rid in visiting:
            raise AssertionError('cycle in recipe dependencies: '+rid)
        if rid in visited:
            return
        visiting.add(rid)
        for dep in sorted(graph[rid]):
            dfs(dep)
        visiting.remove(rid)
        visited.add(rid)
        ordered.append(rid)
    for rid in graph:
        dfs(rid)
    kind_counts=Counter(r['kind'] for r in resources)
    assert kind_counts['site']>0 and kind_counts['utility']>0
    print(f'PASS: DES-004 {len(recipes)} draft nodes, {len(resources)} typed resources, {len({r["family"] for r in recipes})} families; DAG, IDs, fail-closed profiles')
    print('Resource kinds:',dict(sorted(kind_counts.items())))
    print('Draft recipes with unresolved requirements:', dict(sorted(missing.items())))
    if batch:
        print_batch_report(recipes, evidence['profiles'], profile_fields, blockers)
    if chain is not None:
        visited.clear()
        ordered.clear()
        dfs(chain)
        by_id = {recipe['id']: recipe for recipe in recipes}
        report_recipes = [by_id[recipe_id] for recipe_id in ordered]
        print(f'Chain {chain}: {len(report_recipes)} draft operations')
        outside_requirements = {'external_good': set(), 'site': set(), 'utility': set()}
        for recipe in report_recipes:
            for item in recipe['inputs']:
                requirements = outside_requirements.get(item['input_kind'])
                if requirements is not None:
                    requirements.add(item['good_id'])
        for kind, label in (('external_good', 'External goods'), ('site', 'Sites'),
                            ('utility', 'Utilities')):
            values = ', '.join(sorted(outside_requirements[kind])) or 'none'
            print(f'{label}: {values}')
    else:
        report_recipes = sorted(recipes, key=lambda item: (item['family'], item['id']))
    if details or chain is not None:
        for recipe in report_recipes:
            recipe_id = recipe['id']
            version = recipe['version']
            reference = 'yes' if (recipe_id, version) in keys else 'no'
            dependencies = ', '.join(sorted(graph[recipe_id])) or 'none'
            print(f"[{recipe['family']}] {recipe_id} v{version}: "
                  f"draft, cells={cells_by_recipe[recipe_id]}/9, "
                  f"reference={reference}, dependencies={dependencies}")
            print('  Missing:', '; '.join(blockers[recipe_id]))



def print_batch_report(recipes, profiles, profile_fields, blockers):
    by_key = {(profile['recipe_id'], profile['recipe_version']): profile for profile in profiles}
    families = defaultdict(list)
    sources = defaultdict(set)
    models = defaultdict(set)
    for recipe in recipes:
        families[recipe['family']].append(recipe)
    print('Batch report by family:')
    for family, members in sorted(families.items()):
        covered = []
        for recipe in members:
            recipe_id = recipe['id']
            profile = by_key.get((recipe_id, recipe['version']))
            if profile is None:
                continue
            covered.append(recipe_id)
            sources[urlsplit(profile['source']).hostname].add(recipe_id)
            for field in ('equipment_reference', 'oven_reference'):
                equipment = profile.get(field)
                if equipment:
                    models[equipment['model']].add(recipe_id)
                    sources[urlsplit(equipment['source']).hostname].add(recipe_id)
        missing_sources = sorted({recipe['id'] for recipe in members} - set(covered))
        missing_equipment = sorted(recipe['id'] for recipe in members
                                   if 'equipment_profile' in blockers[recipe['id']])
        counts = Counter(field for recipe in members for field in profile_fields
                         if field in blockers[recipe['id']])
        material_gaps = sum(
            any(gap.startswith(('input ', 'output ')) for gap in blockers[recipe['id']])
            for recipe in members
        )
        print(f'  {family}: {len(members)} drafts, {len(covered)} with references')
        print('    No structured reference:', ', '.join(missing_sources) or 'none')
        print('    Missing equipment profile:', ', '.join(missing_equipment) or 'none')
        print(f'    Material gaps: {material_gaps}; missing profile fields: {dict(counts)}')
    for title, groups in (('References by source host', sources), ('Equipment references by model', models)):
        print(f'{title}:')
        for label, recipe_ids in sorted(groups.items()):
            print(f'  {label}: {", ".join(sorted(recipe_ids))}')


if __name__=='__main__':
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--details', action='store_true', help='show checks and gaps for every draft recipe')
    parser.add_argument('--chain', metavar='RECIPE_ID', help='report one recipe and its upstream operations')
    parser.add_argument('--batch', action='store_true', help='group reference coverage and gaps by family')
    args = parser.parse_args()
    validate(details=args.details, chain=args.chain, batch=args.batch)
