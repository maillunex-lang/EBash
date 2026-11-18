#!/usr/bin/env python3
"""Offline EBash handoff audit; no GitHub, no VPS system mutations."""
import json
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECKS = (
    'check_docs.py',
    'test_check_docs.py',
    'check_handoff.py',
    'check_des004_catalog.py',
    'check_des004_pilot_profiles.py',
    'check_des003_balance.py',
    'check_des003_examples.py',
)


def main() -> None:
    for name in CHECKS:
        print(f'== {name} ==', flush=True)
        subprocess.run([sys.executable, str(ROOT / 'scripts' / name)], cwd=ROOT, check=True)
    for path in sorted((ROOT / 'docs' / 'catalogs').glob('*.json')):
        json.loads(path.read_text(encoding='utf-8'))
    print('PASS: all offline handoff checks and catalog JSON parse')


if __name__ == '__main__':
    main()
