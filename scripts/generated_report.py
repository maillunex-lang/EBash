"""Compare a tracked report by default; regenerate only on explicit request."""
import argparse
from pathlib import Path


def check_generated_report(path: Path, text: str) -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--write', action='store_true', help='regenerate the tracked report')
    if parser.parse_args().write:
        path.write_text(text, encoding='utf-8')
    elif path.read_bytes() != text.encode('utf-8'):
        raise SystemExit(f'{path.name} is stale; run this check with --write')
