#!/usr/bin/env python3
"""Deterministic checks for delegated DES-003 working balance, not a production simulator."""
from fractions import Fraction

BASE_QUOTA = 20
EXTRA_QUOTA = 20
EXTRA_FACTOR = Fraction(135, 100)
LOW = Fraction(65, 100)
HIGH = Fraction(150, 100)
MARKDOWN_LOW = Fraction(40, 100)


def city_quote(base_price: int, already_used: int, quantity: int):
    assert base_price > 0 and 0 <= already_used <= BASE_QUOTA and quantity >= 0
    base = min(quantity, BASE_QUOTA - already_used)
    extra = quantity - base
    if extra > EXTRA_QUOTA:
        raise ValueError('Превышен лимит дополнительных закупок')
    return base, extra, base_price * (base + extra * EXTRA_FACTOR)


def price_allowed(price: int, reference: int, *, markdown: bool = False, safe: bool = False, eligible: bool = False):
    if price <= 0 or reference <= 0:
        return False
    floor = MARKDOWN_LOW if markdown and safe and eligible else LOW
    return floor * reference <= price <= HIGH * reference


def score(quality: int, price: int, reference: int, distance: int, radius: int, segment: str):
    weights = {'low': (Fraction(14, 10), Fraction(7, 10)), 'middle': (1, 1), 'high': (Fraction(7, 10), Fraction(14, 10))}
    if segment not in weights or not 1 <= quality <= 100 or min(price, reference, radius) <= 0 or not 0 <= distance <= radius:
        raise ValueError('Недопустимый профиль спроса')
    pw, qw = weights[segment]
    return 100 * qw * Fraction(quality, 100) - 50 * pw * Fraction(price, reference) - 10 * Fraction(distance, radius)


def main():
    assert city_quote(100, 0, 20) == (20, 0, 2000)
    assert city_quote(100, 18, 4) == (2, 2, 470)
    assert city_quote(100, 20, 20) == (0, 20, 2700)
    try:
        city_quote(100, 20, 21)
        raise AssertionError('Exceeded quota was accepted')
    except ValueError:
        pass
    assert price_allowed(65, 100) and price_allowed(150, 100)
    assert not price_allowed(64, 100) and not price_allowed(151, 100)
    assert not price_allowed(40, 100)
    assert price_allowed(40, 100, markdown=True, safe=True, eligible=True)
    assert not price_allowed(40, 100, markdown=True, safe=False, eligible=True)
    assert score(80, 90, 100, 0, 10, 'low') != score(80, 90, 100, 0, 10, 'high')
    assert score(80, 90, 100, 10, 10, 'middle') < score(80, 90, 100, 0, 10, 'middle')
    print('PASS: DES-003 delegated v1 quota, prices, markdown, demand score')


if __name__ == '__main__':
    main()
