#!/usr/bin/env python3
"""Replay isolated DES-004 batches with explicit provisional game profiles."""
from dataclasses import dataclass
from pathlib import Path
import json
from collections import Counter

CATALOG = Path(__file__).resolve().parents[1] / 'docs' / 'catalogs'


@dataclass(frozen=True)
class Stage:
    recipe: str
    inputs: dict[str, int]
    outputs: dict[str, int]
    equipment: str
    min_batch: int
    capacity: int
    energy_wh: int
    operators: int
    quality: str

    def run(self, stock, available_capacity, quality_pass=True):
        if (available_capacity < self.min_batch or self.capacity < self.min_batch
                or not quality_pass or any(stock[k] < n for k, n in self.inputs.items())):
            return False
        stock.subtract(self.inputs)
        stock.update(self.outputs)
        return True


# The batch and resource values below are game assumptions for this replay.
MILL = Stage('R-WHEAT-FLOUR', {'wheat_grain': 10_000}, {'wheat_flour': 7_400},
             'roller mill and sifter', 10_000, 10_000, 500, 1, 'flour accepted')
MIX = Stage('R-BREAD-DOUGH', {'wheat_flour': 7_200, 'water': 5_440, 'salt': 144, 'yeast': 112},
            {'bread_dough': 12_896}, 'Sunmix EASY20', 12_896, 17_732, 250, 1, 'dough accepted')
BAKE = Stage('R-BREAD-LOAF', {'bread_dough': 12_896}, {'bread_loaf': 24},
             'proofer and UNOX oven, two loads', 24, 32, 3_500, 1, 'cooled loaves accepted')
PACK = Stage('R-BREAD-PACK', {'bread_loaf': 24, 'paper_pack': 24}, {'bread_pack': 24},
             'manual packing station', 24, 24, 0, 1, 'pack and closure accepted')
SEPARATE = Stage('R-CREAM', {'raw_milk': 79_900}, {'cream': 7_900, 'skimmed_milk': 72_000},
                 'centrifugal separator', 79_900, 79_900, 750, 1, 'fat assays accepted')
BLEND = Stage('internal dairy flow', {'cream': 2_950, 'skimmed_milk': 37_000},
              {'normalized_mix': 39_950}, 'metering and mixing tank', 39_950, 39_950, 250, 1,
              'blend fat assay accepted')


def check_catalog():
    versions = {'R-WHEAT-FLOUR': 2, 'R-BREAD-DOUGH': 2, 'R-BREAD-LOAF': 1,
                'R-BREAD-PACK': 1, 'R-CREAM': 2}
    recipes = {r['id']: r for r in json.loads((CATALOG / 'recipes.seed.json').read_text())['recipes']}
    for stage in (MILL, MIX, BAKE, PACK, SEPARATE):
        recipe = recipes[stage.recipe]
        assert recipe['version'] == versions[stage.recipe]
        assert recipe['status'] == 'draft_not_executable'
        assert stage.inputs.keys() == {i['good_id'] for i in recipe['inputs'] if i['input_kind'] != 'utility'}
        assert stage.outputs.keys() <= {o['good_id'] for o in recipe['outputs']}


def check_blocked(stage, stock):
    before = stock.copy()
    assert not stage.run(stock, stage.min_batch - 1) and stock == before
    assert not stage.run(stock, stage.capacity, quality_pass=False) and stock == before
    for item in stage.inputs:
        short = before.copy()
        short[item] = stage.inputs[item] - 1
        unchanged = short.copy()
        assert not stage.run(short, stage.capacity) and short == unchanged


def check_bread():
    stock = Counter(wheat_grain=10_000, water=5_440, salt=144, yeast=112, paper_pack=24)
    for stage in (MILL, MIX, BAKE, PACK):
        check_blocked(stage, stock)
        assert stage.run(stock, stage.capacity)
        after = stock.copy()
        assert not stage.run(stock, stage.capacity) and stock == after
    assert +stock == Counter(wheat_flour=200, bread_pack=24)
    assert 10_000 + 5_440 + 144 + 112 == 200 + 2_600 + 1_496 + 24 * 475

    # Release the packaged lot on the next tick and debit it only once.
    packed_at = 3
    cash = 200_000 - 30_000 - 18_000 - 12_000 - 14_000 - 2_400
    for tick in (3, 4, 4):
        if tick > packed_at and stock['bread_pack'] == 24:
            stock['bread_pack'] -= 24
            cash += 24 * 4_500
        assert cash == (123_600 if tick == 3 else 231_600)
    assert +stock == Counter(wheat_flour=200)
    print('PASS: four bread stages, blocked batches, packaging stock and one sale')


def check_dairy():
    stock = Counter(raw_milk=79_900)
    for stage in (SEPARATE, BLEND):
        check_blocked(stage, stock)
        assert stage.run(stock, stage.capacity)
        after = stock.copy()
        assert not stage.run(stock, stage.capacity) and stock == after
    assert +stock == Counter(cream=4_950, skimmed_milk=35_000, normalized_mix=39_950)
    assert sum(stock.values()) == 79_900
    assert 79_900 * 400 == stock['cream'] * 4_000 + stock['skimmed_milk'] * 5 + stock['normalized_mix'] * 300
    print('PASS: milk separation and blending conserve mass and fat')


if __name__ == '__main__':
    check_catalog()
    check_bread()
    check_dairy()
