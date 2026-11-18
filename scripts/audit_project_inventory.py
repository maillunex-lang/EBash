#!/usr/bin/env python3
"""Read-only structural inventory of EBash source files; excludes runtime artifacts."""
import json
from collections import Counter
from source_files import source_files
from pathlib import Path
root=Path(__file__).resolve().parents[1]
scopes=('docs','scripts','server','prototypes','web','.agents')
excluded={'.git','.local','.balance-tools-venv','.godot','__pycache__','node_modules'}
files=[]
for scope in scopes:
    directory=root/scope
    if not directory.exists():continue
    for path in source_files(directory):
        if not path.is_file() or any(p in excluded for p in path.relative_to(root).parts):continue
        files.append(path)
files.sort()
assert files and any(p.name=='game_profiles.go' for p in files)
assert any(p.name=='PilotClient.gd' for p in files)
counts=Counter(p.relative_to(root).parts[0] for p in files)
ext=Counter(p.suffix or '[none]' for p in files)
issues=[]
for p in files:
    if p.stat().st_size==0 and p.suffix in {'.py','.go','.gd','.md','.json'}:
        issues.append(f'empty source: {p.relative_to(root)}')
    if p.suffix=='.json':
        try:json.loads(p.read_text(encoding='utf-8'))
        except (ValueError,UnicodeError) as e:issues.append(f'bad JSON: {p.relative_to(root)}: {e}')
    if p.suffix in {'.py','.go','.gd','.md'}:
        try:p.read_text(encoding='utf-8')
        except UnicodeError as e:issues.append(f'bad UTF-8: {p.relative_to(root)}: {e}')
assert not issues,'; '.join(issues)
print('PASS: structural inventory',len(files),'files;',dict(sorted(counts.items())))
print('Extensions:',dict(sorted(ext.items())))
print('Excluded: .git, server/.local, .balance-tools-venv, .godot, __pycache__, node_modules')
print('Scope: file readability, JSON parse, UTF-8 and zero-size; NOT proof of runtime correctness')
