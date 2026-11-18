#!/usr/bin/env python3
"""Negative checks of Markdown validation on an isolated, disposable copy."""
from pathlib import Path
import os
from source_files import source_files
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / 'scripts' / 'check_docs.py'


def run(root: Path) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    env['EBASH_DOC_ROOT'] = str(root)
    return subprocess.run(['python3', str(CHECK)], env=env, text=True, capture_output=True, check=False)


def main() -> None:
    with tempfile.TemporaryDirectory(prefix='ebash-docs-audit-') as tmp:
        root = Path(tmp)
        for src in source_files(ROOT):
            dst = root / src.relative_to(ROOT)
            dst.parent.mkdir(parents=True, exist_ok=True)
            if src.suffix == '.md':
                shutil.copy2(src, dst)
            else:
                dst.touch()
        baseline = run(root)
        assert baseline.returncode == 0, baseline.stdout + baseline.stderr
        task = root / 'docs/tasks/DES-004.md'
        original = task.read_text()
        task.write_text(original.replace('Статус: в работе.', 'Статус: заблокирована.', 1))
        result = run(root)
        assert result.returncode != 0 and 'inconsistent queue status' in result.stdout, result.stdout
        task.write_text(original)
        readme = root / 'README.md'
        original_readme = readme.read_text()
        readme.write_text(original_readme + '\n[invalid](docs/DOES_NOT_EXIST.md)\n')
        result = run(root)
        assert result.returncode != 0 and 'broken link' in result.stdout, result.stdout
        readme.write_text(original_readme)
        decision = root / 'docs/decisions/001-foundation.md'
        original_decision = decision.read_text()
        decision.write_text(original_decision.replace('| D-153 |', '| D-155 |', 1))
        result = run(root)
        assert result.returncode != 0 and 'not continuous' in result.stdout, result.stdout
        decision.write_text(original_decision)
        assert run(root).returncode == 0
    print('PASS: Markdown validator detects inconsistent task, broken link and missing decision number on isolated copies')


if __name__ == '__main__':
    main()
