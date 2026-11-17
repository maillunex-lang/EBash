#!/usr/bin/env python3
"""DES-003: arithmetic invariants for illustrative GDD scenarios, not game engine tests."""
from fractions import Fraction


def buy(wanted: int, budget: int, stock: int, price: int) -> tuple[int, int, int]:
    assert wanted >= 0 and budget >= 0 and stock >= 0 and price > 0
    sold = min(wanted, budget // price, stock)
    return sold, budget - sold * price, stock - sold


def split_quota(quantity: int, quota_left: int, base_price: int, extra_price: int) -> tuple[int, int, int]:
    assert min(quantity, quota_left, base_price, extra_price) >= 0
    base = min(quantity, quota_left)
    extra = quantity - base
    return base, extra, base * base_price + extra * extra_price


def main() -> None:
    assert buy(10, 80, 20, 10) == (8, 0, 12)  # M01
    assert buy(10, 80, 5, 10) == (5, 30, 0)  # M02
    assert buy(12, 90, 7, 10) == (7, 20, 0)  # M15
    assert buy(10, 40, 10, 10)[0] == 4  # M16, after 60 already spent
    assert 1000 - 10 * 40 - 50 + 8 * 70 == 1110  # S01
    assert split_quota(12, 10, 40, 60) == (10, 2, 520)  # S02
    assert split_quota(5, 3, 40, 60) == (3, 2, 240)  # S16
    # S17/S20: replay must return a recorded result, not execute a second purchase.
    operations = {}
    balance, inventory, quota_used = 1000, 0, 0
    for _ in range(2):
        operation_id = 'city-order-001'
        if operation_id not in operations:
            charge = 4 * 40
            balance -= charge
            inventory += 4
            quota_used += 4
            operations[operation_id] = (charge, 4)
    assert (balance, inventory, quota_used) == (840, 4, 4)
    # M03: two stores cannot spend the same group budget twice.
    first, remaining, _ = buy(6, 100, 6, 10)
    second, final, _ = buy(8, remaining, 8, 10)
    assert (first, second, final) == (6, 4, 0)
    # Provenance conservation for a pure transfer: A19/A20.
    origin_stock = 10
    transferred = 4
    assert origin_stock - transferred + transferred == origin_stock
    # D-107 arithmetic demonstration uses exact rationals, not float.
    assert Fraction(1, 3) + Fraction(2, 3) == 1
    print('PASS: DES-003 illustrative arithmetic, quota replay and stock conservation')


if __name__ == '__main__':
    main()
