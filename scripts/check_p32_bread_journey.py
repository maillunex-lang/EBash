#!/usr/bin/env python3
"""P3.2: deterministic financial and tick acceptance for the provisional bread chain.

This is a game scenario, NOT evidence to unblock DES-004 recipes.
"""
from collections import Counter
from dataclasses import dataclass
from pathlib import Path
import json

ROOT = Path(__file__).resolve().parents[1]
RECIPES = {r['id']: r for r in json.loads((ROOT/'docs/catalogs/recipes.seed.json').read_text())['recipes']}
EVIDENCE = json.loads((ROOT/'docs/catalogs/pilot-evidence.json').read_text())
CHAIN = ('R-WHEAT-FLOUR', 'R-BREAD-DOUGH', 'R-BREAD-LOAF')


def check_draft_guard():
    assert EVIDENCE['status'] == 'reference_only'
    indexed = {p['recipe_id']: p for p in EVIDENCE['profiles']}
    for name in CHAIN:
        assert RECIPES[name]['status'] == 'draft_not_executable', name
        assert indexed[name]['unverified'], name
    assert 'measured_yield' in indexed['R-BREAD-LOAF']['unverified']
    assert 'energy' in indexed['R-BREAD-DOUGH']['unverified']


@dataclass
class Journey:
    delivery_lag: int = 0
    cash: int = 200000  # kopecks

    def __post_init__(self):
        self.stock = Counter()
        self.shipment = None
        self.milled = self.mixed = self.baked = self.sold = False
        self.events = set()
        self.costs = Counter()
        self.revenue = 0

    def debit(self, event, amount):
        if event in self.events:
            return False
        self.events.add(event)
        self.cash -= amount
        self.costs[event] = amount
        return True

    def start(self):
        self.debit('grain', 30000)
        self.debit('ingredients', 18000)
        self.stock.update(wheat_grain=10000, water=5440, salt=144, yeast=112)

    def tick(self, t):
        if t >= 1 and not self.milled:
            assert self.stock['wheat_grain'] == 10000
            self.stock['wheat_grain'] -= 10000
            self.milled = True
            self.debit('delivery', 12000)
            self.shipment = 1 + self.delivery_lag
        if self.shipment is not None and t >= self.shipment and 'received' not in self.events:
            self.events.add('received')
            self.stock['wheat_flour'] += 7400
            self.available_from = t + 1
        if t >= 2 and 'received' in self.events and t >= self.available_from and not self.mixed:
            for ingredient, amount in (('wheat_flour', 7200), ('water', 5440), ('salt', 144), ('yeast', 112)):
                assert self.stock[ingredient] >= amount
                self.stock[ingredient] -= amount
            self.stock['bread_dough'] += 12896
            self.mixed = True
            self.debit('operations', 14000)
        if self.mixed and not self.baked:
            self.stock['bread_dough'] -= 12896
            self.stock['bread_loaf'] += 24
            self.baked = True
            self.baked_at = t
        if self.baked and t > self.baked_at and not self.sold:
            self.stock['bread_loaf'] -= 24
            self.cash += 24 * 4500
            self.revenue += 24 * 4500
            self.sold = True


def check_journey(lag):
    game = Journey(delivery_lag=lag)
    game.start()
    assert game.cash == 152000
    for t in range(5 + lag):
        before = (game.cash, game.stock.copy(), game.revenue)
        game.tick(t)
        after = (game.cash, game.stock.copy(), game.revenue)
        game.tick(t)  # retry of the same tick must not change money or inventory
        assert (game.cash, game.stock, game.revenue) == after, (lag, t)
        if t == 2 + lag:
            assert game.baked and not game.sold
            assert game.cash == 126000
        if t < 2 + lag:
            assert not game.baked
    assert game.sold and game.baked_at == 2 + lag
    assert game.cash == 234000 and game.revenue == 108000
    assert +game.stock == Counter(wheat_flour=200)
    assert sum(game.costs.values()) == 74000
    assert game.cash - 200000 == 34000
    # The unconsumed flour carries 11.35 rubles of cost, so cash change != sales profit.
    assert (game.revenue - sum(game.costs.values())) == 34000
    print(f'PASS: delivery lag {lag}, bake tick {game.baked_at}, sale tick {game.baked_at+1}, cash 2340 RUB')


if __name__ == '__main__':
    check_draft_guard()
    check_journey(0)
    check_journey(1)
    print('P3.2 PASS: draft guard, next-tick sale, delayed delivery, idempotency, mass and cash')
