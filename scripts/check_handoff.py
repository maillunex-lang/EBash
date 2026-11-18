#!/usr/bin/env python3
"""Assert handoff claims against local task/decision/catalog data."""
import json
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding='utf-8')


def main() -> None:
    status = read('docs/STATUS.md')
    handoff = read('docs/HANDOFF.md')
    catalog = read('docs/catalogs/INDEX.md')
    recipes = json.loads(read('docs/catalogs/recipes.seed.json'))['recipes']
    resources = json.loads(read('docs/catalogs/resources.seed.json'))['resources']
    assert str(len(recipes)) in catalog and str(len(resources)) in catalog, 'Catalog index counts are stale'
    assert '(roadmap/stages.md)' in status and '(STATUS.md)' in handoff, 'Missing handoff route'
    assert all('GitHub' in x for x in (read('AGENTS.md'), status, handoff)), 'Publication boundary missing'
    backlog = read('docs/roadmap/implementation-backlog.md')
    stages = read('docs/roadmap/stages.md')
    packages = re.split(r'^## (E-\d{3})$', backlog, flags=re.MULTILINE)
    ids = packages[1::2]
    assert len(ids) == len(set(ids)) and set(ids) >= {f'E-{i}' for i in range(101, 109)}, 'Missing or duplicate package'
    for package, body in zip(ids, packages[2::2]):
        assert f'(implementation-backlog.md#{package.lower()})' in stages, f'No stage points to {package}'
        for field in ('- Открыть [', '- Приёмка', '- Проверка'):
            assert field in body, f'{package} lacks {field}'
    print(f'PASS: handoff routes and catalog index ({len(recipes)} nodes, {len(resources)} resources)')



if __name__ == '__main__':
    main()
