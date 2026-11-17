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
    queue = read('docs/tasks/INDEX.md')
    catalog = read('docs/catalogs/INDEX.md')
    decisions = re.findall(r'^\| (D-\d{3}) \|', read('docs/decisions/001-foundation.md'), re.M)
    recipes = json.loads(read('docs/catalogs/recipes.seed.json'))['recipes']
    resources = json.loads(read('docs/catalogs/resources.seed.json'))['resources']
    assert f'**{len(decisions)} решения**' in status, 'Incorrect decision count in STATUS'
    assert 'D-149' in handoff, 'Delegation must be mentioned in handoff'
    assert all(x in status for x in ('DES-004', 'DES-005', 'DES-007'))
    assert 'DES-004](DES-004.md) | Каталоги городов и производственных цепочек | в работе' in queue
    assert 'DES-005](DES-005.md) | Финансы, персонал и инфраструктура | готова к проектированию' in queue
    assert 'DES-007](DES-007.md) | Android-интерфейс и обучение | готова к проектированию' in queue
    assert 'DES-006](DES-006.md) | Рост городов, бюджет и выборы | заблокирована' in queue
    assert 'DES-008](DES-008.md) | Архитектура и задачи реализации | заблокирована' in queue
    for text in (status, handoff, catalog):
        assert str(len(recipes)) in text and str(len(resources)) in text, 'Catalog counts in handoff are stale'
    assert all(r['status'] == 'draft_not_executable' for r in recipes), 'Executable recipe in claimed draft set'
    assert all(r['min_batch'] is None for r in recipes), 'Handoff claims missing numeric profiles'
    assert all(r['unit'] is None for r in resources), 'Units now present, handoff must be updated'
    assert all('GitHub' in x for x in (read('AGENTS.md'), status, handoff)), 'Publication boundary missing'
    print(f'PASS: handoff matches {len(decisions)} decisions, {len(recipes)} draft nodes and {len(resources)} resources, task statuses and VPS-only policy')


if __name__ == '__main__':
    main()
