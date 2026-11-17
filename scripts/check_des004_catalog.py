#!/usr/bin/env python3
"""Validate draft catalog identifiers, dependency graph and fail-closed readiness."""
import json
from collections import Counter
from pathlib import Path


def validate():
    root=Path(__file__).resolve().parent.parent/'docs'/'catalogs'
    data=json.loads((root/'recipes.seed.json').read_text())
    resources=json.loads((root/'resources.seed.json').read_text())['resources']
    recipes=data['recipes']
    ids=[r['id'] for r in recipes]
    assert len(ids)==len(set(ids)) and len(ids)>=100
    assert all(r['version']==1 and r['status']=='draft_not_executable' for r in recipes)
    assert all(r['inputs'] and r['outputs'] and r['source_status']=='requires_verification' for r in recipes)
    assert all(r['min_batch'] is None and r['equipment_profile'] is None for r in recipes)
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
    visited=set();visiting=set()
    def dfs(rid):
        if rid in visiting:
            raise AssertionError('cycle in recipe dependencies: '+rid)
        if rid in visited:
            return
        visiting.add(rid)
        for dep in graph[rid]:
            dfs(dep)
        visiting.remove(rid);visited.add(rid)
    for rid in graph:
        dfs(rid)
    kind_counts=Counter(r['kind'] for r in resources)
    assert kind_counts['site']>0 and kind_counts['utility']>0
    print(f'PASS: DES-004 {len(recipes)} draft nodes, {len(resources)} typed resources, {len({r["family"] for r in recipes})} families; DAG, IDs, fail-closed profiles')
    print('Resource kinds:',dict(sorted(kind_counts.items())))


if __name__=='__main__':
    validate()
