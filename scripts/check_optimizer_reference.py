"""Independent CP-SAT reference for a two-mill DES-002 resource scenario.

This is a solver smoke test, not a replacement for Go's game optimizer.
"""
from ortools.sat.python import cp_model


def main():
    model = cp_model.CpModel()
    a = model.new_int_var(0, 2, 'mill_a_batches')
    b = model.new_int_var(0, 2, 'mill_b_batches')
    model.add(10000 * (a + b) <= 30000)
    model.add(500 * (a + b) <= 1500)
    model.add(12000 * (a + b) <= 36000)
    model.maximize(a + b)
    solver = cp_model.CpSolver()
    solver.parameters.num_search_workers = 1
    solver.parameters.max_time_in_seconds = 2
    status = solver.solve(model)
    if status != cp_model.OPTIMAL or solver.objective_value != 3:
        raise RuntimeError(f'expected exactly 3 batches, got {solver.status_name(status)}')
    print(f'CP-SAT reference PASS: 3 batches (a={solver.value(a)}, b={solver.value(b)})')


if __name__ == '__main__':
    main()
