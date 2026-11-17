"""Validate this repository's inline Markdown links, decisions and design task graph."""
from pathlib import Path
import re
import sys
from urllib.parse import unquote, urlsplit

ROOT = Path(__file__).resolve().parents[1]
errors = []
files = sorted(ROOT.rglob('*.md'))
files = [p for p in files if '.git' not in p.relative_to(ROOT).parts]
texts = {p: p.read_text(encoding='utf-8') for p in files}

def anchors(text):
    used = {}
    result = set()
    for heading in re.findall(r'^#{1,6}\s+(.+)$', text, re.M):
        slug = re.sub(r'[^\w\- ]', '', heading.lower()).replace(' ', '-')
        n = used.get(slug, 0)
        used[slug] = n + 1
        result.add(slug if not n else f'{slug}-{n}')
    return result

links = 0
external = set()
for p, text in texts.items():
    for target in re.findall(r'!?\[[^\]]*\]\(([^)]+)\)', text):
        links += 1
        u = urlsplit(target)
        if u.scheme:
            external.add(target)
            continue
        dest = (p.parent / unquote(u.path)).resolve() if u.path else p
        if not dest.is_relative_to(ROOT) or not dest.is_file():
            errors.append(f'{p.relative_to(ROOT)}: broken link {target}')
        elif u.fragment and dest.suffix == '.md' and unquote(u.fragment) not in anchors(dest.read_text()):
            errors.append(f'{p.relative_to(ROOT)}: missing anchor {target}')

registry = texts[ROOT / 'docs/decisions/001-foundation.md']
ids = re.findall(r'^\| (D-\d{3}) \|', registry, re.M)
if len(ids) != len(set(ids)):
    errors.append('Duplicate decision IDs')
for p, text in texts.items():
    for id in re.findall(r'\bD-\d{3}\b', text):
        if id not in ids:
            errors.append(f'{p.relative_to(ROOT)}: unknown decision {id}')

# Every document is discoverable from README via actual file links.
seen = set()
def visit(p):
    if p in seen or p not in texts:
        return
    seen.add(p)
    for target in re.findall(r'\[[^\]]*\]\(([^)]+)\)', texts[p]):
        u = urlsplit(target)
        if not u.scheme:
            visit((p.parent / unquote(u.path)).resolve())
visit(ROOT / 'README.md')
for p in texts.keys() - seen:
    errors.append(f'Undiscoverable document: {p.relative_to(ROOT)}')

tasks = {p.stem: text for p, text in texts.items() if re.fullmatch(r'DES-\d{3}', p.stem)}
graph = {}
index = texts[ROOT / 'docs/tasks/INDEX.md']
for id, text in tasks.items():
    for section in ('Цель', 'Читать', 'Зависимости', 'Объём', 'Приёмка', 'Проверка', 'Результат'):
        if f'## {section}\n' not in text:
            errors.append(f'{id}: missing section {section}')
    goal = text.split('## Цель\n', 1)[-1].split('\n## ', 1)[0].strip()
    scope = text.split('## Объём\n', 1)[-1].split('\n## ', 1)[0]
    if goal and goal in scope:
        errors.append(f'{id}: section Объём repeats Цель')
    match = re.search(r'^Статус: (.+)\.$', text, re.M)
    row = next((r for r in index.splitlines() if r.startswith(f'| [{id}]')), '')
    if not match or match[1] not in row:
        errors.append(f'{id}: inconsistent queue status')
    dep_section = text.split('## Зависимости\n', 1)[-1].split('\n## ', 1)[0]
    graph[id] = re.findall(r'\[(DES-\d{3})\]', dep_section)
    if set(graph[id]) != set(re.findall(r'DES-\d{3}', row)) - {id}:
        errors.append(f'{id}: inconsistent queue dependencies')
    for dep in graph[id]:
        if dep not in tasks:
            errors.append(f'{id}: missing dependency {dep}')

def walk(id, path):
    if id in path:
        errors.append('Dependency cycle: ' + ' -> '.join(path + [id]))
        return
    for dep in graph.get(id, []):
        walk(dep, path + [id])
for id in graph:
    walk(id, [])
if errors:
    print('\n'.join(errors))
    sys.exit(1)
print(f'OK: {len(files)} documents, {links} inline links, {len(ids)} unique decisions, {len(tasks)} tasks; no unreachable documents or dependency cycles.')
print(f'External URLs: {len(external)} (HTTP availability is not checked by this offline validator).')
