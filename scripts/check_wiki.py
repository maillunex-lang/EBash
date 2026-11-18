#!/usr/bin/env python3
"""Regression checks for the read-only EBash encyclopedia, no web/network needed."""
import json
import shutil
import subprocess
from html.parser import HTMLParser
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
WEB = ROOT / 'web' / 'wiki'
SOURCE = ROOT / 'docs' / 'catalogs'

class PageParser(HTMLParser):
    def __init__(self):
        super().__init__()
        self.ids = set()
        self.scripts = []
        self.links = []
    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if 'id' in attrs:
            assert attrs['id'] not in self.ids, 'Duplicate DOM ID'
            self.ids.add(attrs['id'])
        if tag == 'script':
            self.scripts.append(attrs.get('src'))
        if tag == 'a':
            self.links.append(attrs.get('href'))


def main():
    catalog = json.loads((WEB / 'catalog.json').read_text(encoding='utf-8'))
    sources = {x['id'] for x in json.loads((SOURCE/'resources.seed.json').read_text())['resources']}
    drafts = {x['id'] for x in json.loads((SOURCE/'recipes.seed.json').read_text())['recipes']}
    approved = {x['id'] for x in json.loads((SOURCE/'game-production-profiles.v1.json').read_text())['profiles']}
    assert {x['id'] for x in catalog['goods']} == sources
    good_ids = {x['id'] for x in catalog['goods']}
    assert len(catalog['goods']) == len(good_ids)
    got_drafts = [x for x in catalog['recipes'] if x['status'] == 'draft']
    got_approved = [x for x in catalog['recipes'] if x['status'] == 'approved']
    assert {x['source_id'] for x in got_drafts} == drafts
    assert {x['id'][len('approved:'):] for x in got_approved} == approved
    assert len(catalog['recipes']) == len(drafts)+len(approved)
    for recipe in got_drafts:
        assert all(x['quantity'] is None for p in recipe['inputs']+recipe['outputs'] for x in [p]), 'Draft quantities must not be fabricated'
        assert 'energy_wh' not in recipe and 'cost_kopecks' not in recipe
    for recipe in catalog['recipes']:
        assert recipe['inputs'] and recipe['outputs']
        assert all(x['good'] in good_ids for x in recipe['inputs']+recipe['outputs'])
    parser = PageParser()
    parser.feed((WEB / 'index.html').read_text(encoding='utf-8'))
    assert {'search','family','results','article','featured-path','metrics'}.issubset(parser.ids)
    assert './wiki.js' in parser.scripts and '../' in parser.links
    landing = (ROOT / 'web' / 'landing' / 'index.html').read_text(encoding='utf-8')
    assert landing.count('href="./wiki/"') >= 2, 'Landing must link to the encyclopedia'
    js = (WEB / 'wiki.js').read_text(encoding='utf-8')
    assert "fetch('./catalog.json'" in js and '[data-recipe]' in js and 'renderRecipe' in js and 'renderGood' in js
    if shutil.which('node'):
        subprocess.run(['node','--check',str(WEB/'wiki.js')],check=True)
    print(f'PASS: wiki {len(good_ids)} goods, {len(got_approved)} approved recipes, {len(got_drafts)} explicitly non-executable drafts, HTML and JS')

if __name__=='__main__':
    main()
