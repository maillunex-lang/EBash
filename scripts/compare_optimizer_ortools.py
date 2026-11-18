"""CP-SAT oracle for bounded two-mill scenarios received as JSON on stdin."""
import json
import sys
from ortools.sat.python import cp_model

cases = json.load(sys.stdin)
answers = []
for case in cases:
    model = cp_model.CpModel()
    a = model.new_int_var(0, min(2, case["requests"][0]), "a")
    b = model.new_int_var(0, min(2, case["requests"][1]), "b")
    total = a + b
    model.add(total * 10000 <= case["grain"])
    model.add(total * 500 <= case["energy"])
    model.add(total * 12000 <= case["cash"])
    model.add(total <= case["capacity"])
    # One worker is required per active line, independent of its batch count.
    used_a = model.new_bool_var("used_a")
    used_b = model.new_bool_var("used_b")
    model.add(a > 0).only_enforce_if(used_a)
    model.add(a == 0).only_enforce_if(used_a.Not())
    model.add(b > 0).only_enforce_if(used_b)
    model.add(b == 0).only_enforce_if(used_b.Not())
    model.add(used_a + used_b <= case["workers"])
    model.maximize(total)
    solver = cp_model.CpSolver()
    solver.parameters.num_search_workers = 1
    solver.parameters.max_time_in_seconds = 2
    result = solver.solve(model)
    if result != cp_model.OPTIMAL:
        raise RuntimeError("oracle did not prove optimality")
    best_total = int(solver.objective_value)
    model.add(total == best_total)
    r1, r2 = case["requests"]
    request_total = r1 + r2
    quota_pool = min(case["grain"] // 10000, request_total)
    deviation = []
    for name, count, request, other in [("a", a, r1, r2), ("b", b, r2, r1)]:
        diff = model.new_int_var(-1000, 1000, f"diff_{name}")
        model.add(diff == count * request_total - quota_pool * request)
        abs_diff = model.new_int_var(0, 1000, f"abs_{name}")
        model.add_abs_equality(abs_diff, diff)
        deviation.append(abs_diff * other)
    deviation_total = sum(deviation)
    model.minimize(deviation_total)
    result = solver.solve(model)
    if result != cp_model.OPTIMAL:
        raise RuntimeError("oracle did not prove minimum quota deviation")
    best_deviation = int(solver.objective_value)
    model.add(deviation_total == best_deviation)
    model.maximize(a)  # stable-ID tie break
    result = solver.solve(model)
    if result != cp_model.OPTIMAL:
        raise RuntimeError("oracle did not prove stable-ID tie break")
    answers.append([int(solver.value(a)), int(solver.value(b))])
json.dump(answers, sys.stdout)
