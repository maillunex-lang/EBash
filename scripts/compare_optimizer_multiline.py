"""Independent exhaustive oracle for bounded DES-002 shared-resource groups.

No production database, no Go optimizer imports, no external packages.
"""
import itertools
import json
import sys
from fractions import Fraction

def solve(c):
    lines = c["lines"]
    stock = c["stock"]
    claims = {resource: sum(line["inputs"].get(resource, 0) * line["request"] for line in lines) for resource in stock}
    quotas = {
        resource: {i: Fraction(min(stock[resource], total) * line["inputs"].get(resource, 0) * line["request"], total) if total else Fraction(0)
                   for i, line in enumerate(lines)}
        for resource, total in claims.items()
    }
    best = None
    best_key = None
    for plan in itertools.product(*(range(min(line["request"], line["limit"]) + 1) for line in lines)):
        if sum(plan[i] * lines[i]["energy"] for i in range(len(lines))) > c["energy"]:
            continue
        if sum(plan[i] * lines[i]["cost"] for i in range(len(lines))) > c["cash"]:
            continue
        if sum(lines[i]["workers"] for i, count in enumerate(plan) if count) > c["workers"]:
            continue
        if any(sum(plan[i] * lines[i]["inputs"].get(resource, 0) for i in range(len(lines))) > amount for resource, amount in stock.items()):
            continue
        if any(sum(plan[i] for i, line in enumerate(lines) if line["equipment"] == equipment) > capacity for equipment, capacity in c["equipment"].items()):
            continue
        deviation = sum(
            abs(Fraction(plan[i] * n) - quotas[resource][i]) / (n * line["request"])
            for i, line in enumerate(lines) for resource, n in line["inputs"].items()
        )
        key = (sum(plan), -deviation, tuple(plan))
        if best_key is None or key > best_key:
            best_key, best = key, plan
    return list(best)

json.dump([solve(c) for c in json.load(sys.stdin)], sys.stdout)
