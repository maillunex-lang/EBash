#!/usr/bin/env python3
"""Build a reproducible, read-only DES-004 activation gap report."""
import json
from collections import Counter, defaultdict
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CAT = ROOT / "docs/catalogs"
recipes = json.loads((CAT / "recipes.seed.json").read_text())["recipes"]
evidence = json.loads((CAT / "pilot-evidence.json").read_text())["profiles"]
game = json.loads((CAT / "game-production-profiles.v1.json").read_text())["profiles"]
reference_keys = {(p["recipe_id"], p["recipe_version"]) for p in evidence}
approved_keys = {(p["recipe_id"], p["recipe_version"]) for p in game}
assert len(reference_keys) == len(recipes) == 105
assert approved_keys <= reference_keys
fields = ("min_batch", "equipment_profile", "energy_profile", "workforce_profile", "quality_profile")
by_family = defaultdict(list)
for recipe in recipes:
    by_family[recipe["family"]].append(recipe)
out = [
    "# DES-004: матрица готовности рецептов к активации",
    "",
    "Автоматически формируется скриптом `python3 scripts/report_des004_activation.py`.",
    "Статус: **аудит, не разрешение исполнения**. Все 105 черновых операций остаются заблокированными.",
    "Три утверждённых игровых профиля хранятся отдельно, не переводят исходные черновики в исполняемый статус.",
    "",
    "## Сводка по отраслям",
    "",
    "| Семейство | Черновики | Есть справочная запись | Отдельные игровые профили | Не хватает количества/единиц |",
    "| --- | ---: | ---: | ---: | ---: |",
]
for family, members in sorted(by_family.items()):
    refs = sum((r["id"], r["version"]) in reference_keys for r in members)
    profiles = sum((r["id"], r["version"]) in approved_keys for r in members)
    material = sum(any(i.get("quantity") is None or i.get("unit") is None for i in r["inputs"] + r["outputs"] if i.get("input_kind") not in ("site", "utility")) for r in members)
    out.append(f"| {family} | {len(members)} | {refs} | {profiles} | {material} |")
out.extend([
    "",
    "## Ворота активации",
    "",
    "1. Уточнить единицы и нормы каждого **складского** входа и выхода; участки и энергию не превращать в расходуемые товары.",
    "2. Заполнить минимальную партию, оборудование/производительность, энергию, работников и качество с версией и происхождением нормы.",
    "3. Согласовать совместные выходы и потери, особенно отруби, сыворотку и нефтяные фракции. Исключить выпуск бесплатных побочных продуктов.",
    "4. Проверить сырьевые цепочки, ограничения 3×3, складскую доступность, резервы, себестоимость и торговые тики.",
    "5. Подключать к исполняемому реестру только отдельно проверенный игровой профиль. Исторические черновики сохранять.",
    "",
    "## Первая очередь: зерновая отрасль",
    "",
    "| Рецепт | Версия | Игровой профиль | Незакрытые параметры | Складские количества |",
    "| --- | ---: | --- | --- | --- |",
])
for r in by_family["grain"]:
    key = (r["id"], r["version"])
    missing = ", ".join(f for f in fields if r.get(f) is None) or "нет"
    material = [i["good_id"] for i in r["inputs"] + r["outputs"] if i.get("input_kind") not in ("site", "utility") and (i.get("quantity") is None or i.get("unit") is None)]
    out.append(f'| {r["id"]} | {r["version"]} | {"отдельный утверждённый" if key in approved_keys else "отсутствует"} | {missing} | {", ".join(material) or "заполнены"} |')
out.extend([
    "",
    "## Ограничения текущего аудита",
    "",
    "Наличие справочной записи не означает подтверждения технологических норм. Полная проверка материального баланса всех 105 операций невозможна до заполнения количеств. Три отдельных хлебных игровых профиля не покрывают весь зерновой каталог.",
    "",
])
target = CAT / "activation-audit.md"
target.write_text("\n".join(out), encoding="utf-8")
print(f"PASS: {len(recipes)} recipes, {len(by_family)} families, {len(approved_keys)} separate game profiles; wrote {target.relative_to(ROOT)}")
