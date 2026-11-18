"""Iterate project files without walking generated databases and tool caches."""
import os
from pathlib import Path


def source_files(root: Path):
    for directory, folders, files in os.walk(root):
        folders[:] = [
            name for name in folders
            if (not name.startswith('.') or name in {'.agents', '.github'})
            and name not in {'node_modules', 'venv', '__pycache__'}
        ]
        for name in files:
            yield Path(directory) / name
