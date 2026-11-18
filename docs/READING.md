# Маршруты чтения для ИИ-исполнителя

Для одного пакета, например E-106, достаточно команды `grep -A7 "^## E-106$" docs/roadmap/implementation-backlog.md`. Если известен ID E-101…E-108, открой только раздел с этим ID в [backlog](roadmap/implementation-backlog.md) и перечисленные там файлы. Для проектной задачи DES-XXX открой **один** `docs/tasks/DES-XXX.md`. Реестр решений читать точечно по `D-XXX` через `grep -n`. Не загружай [001-foundation.md](../docs/decisions/001-foundation.md), [DES-004-history.md](../docs/tasks/DES-004-history.md) или все JSON целиком без необходимости.

| Тип задачи | Первые файлы | Минимальная проверка |
| --- | --- | --- |
| Тики, атомарность, API | [store.go](../server/store.go), [integration_test.go](../server/integration_test.go), нужный Go-модуль | `cd server && go test ./...` |
| Оптимизатор и стадии | [E-102](roadmap/implementation-backlog.md#e-102) или [E-103](roadmap/implementation-backlog.md#e-103) | Из `server/`: `go test -run TestGame ./...` |
| Склады и торговля | [E-104](roadmap/implementation-backlog.md#e-104) | Из корня: `bash server/check_p32_postgres.sh` |
| Рецепт или отрасль | [DES-004](tasks/DES-004.md), [индекс каталога](catalogs/INDEX.md); только нужный ID | `python3 scripts/check_des004_catalog.py --chain R-BREAD-LOAF` (заменить ID) |
| DES-003, цены и спрос | [DES-003](tasks/DES-003.md), [market.md](../docs/gdd/market.md) | `python3 scripts/check_des003_balance.py` |
| Энциклопедия лендинга | [index.html](../web/landing/index.html), [index.html](../web/wiki/index.html), [wiki.js](../web/wiki/wiki.js), [build_wiki_catalog.py](../scripts/build_wiki_catalog.py) | `python3 scripts/check_wiki.py`; Nginx `/games/bash/wiki/` |
| Республика и компания | [Region.gd](../prototypes/godot/Region.gd), [города](../prototypes/godot/CityCatalog.gd) | `region_smoke.gd` |
| Город, камера, поиск | [Main.gd](../prototypes/godot/Main.gd), [генератор](../prototypes/godot/District.gd) | `smoke.gd` и `region_smoke.gd` (только игровой маршрут, D-192) |
| Настройки | [меню](../prototypes/godot/Launcher.gd), [сохранение](../prototypes/godot/Preferences.gd) | `preferences_smoke.gd` в двух процессах, см. ниже |
| Стройка | [сервер](../server/construction.go), [карта](../prototypes/godot/Main.gd) | `construction-smoke` |
| Сеть и предприятие | [клиент API](../prototypes/godot/PilotClient.gd), [экран](../prototypes/godot/PilotPanel.gd) | `profile-demo-smoke` (тестовый игровой мир, не UI-стенд; D-192) |
| Сборка или телефон | [сборка APK](../.agents/skills/ebash-android-build/SKILL.md), [параметры мира](roadmap/p3-public-path.md) | Проверка подписи, ручной запуск по [E-106](roadmap/implementation-backlog.md#e-106) |
| Финансы/город/архитектура | Только соответствующий [DES-005.md](../docs/tasks/DES-005.md), [DES-006](tasks/DES-006.md) или [DES-008](tasks/DES-008.md) | Профильные проверки и `python3 scripts/check_all.py` |

**Перед работой:** `git status --short`, [AGENTS](../AGENTS.md). **После:** целевые тесты, `python3 scripts/check_all.py`, `git diff --check`; обновить только профильный протокол и краткий [STATUS](STATUS.md) при изменении готовности. [Аудит рисков](audits/2026-11-18.md) читать при работе с E-101–104, не при каждой UI-задаче.

## Рецепты, мебель, магазины и аудит без лишнего чтения

| Запрос | Открыть только | Проверить |
| --- | --- | --- |
| Мебельный завод | [furniture.proposal.json](../docs/catalogs/furniture.proposal.json) | `python3 scripts/check_des004_furniture.py` |
| Магазины и розничные SKU | [retail-channels.proposal.json](../docs/catalogs/retail-channels.proposal.json) | `python3 scripts/check_des004_retail_channels.py` |
| Зерновые сценарии | [grain-scenarios.proposal.json](../docs/catalogs/grain-scenarios.proposal.json) | `python3 scripts/check_des004_grain_scenarios.py` |
| Экономика зерна | [grain-economy.audit.json](../docs/catalogs/grain-economy.audit.json) | `python3 scripts/check_des004_grain_economy.py` |
| Аудит всего дерева исходников | [audit_project_inventory.py](../scripts/audit_project_inventory.py), [2026-11-18-full-project-followup.md](../docs/audits/2026-11-18-full-project-followup.md) | `python3 scripts/check_all.py` |

**Не читать без необходимости:** `server/.local` (тестовые БД и артефакты), большие исторические GDD, [DES-004-history.md](../docs/tasks/DES-004-history.md), `.balance-tools-venv`. Новый исполнитель не должен считать неисполняемые предложения действующими рецептами.


## Запуск клиентских проверок на Lunex

Из корня рабочей копии:

```bash
export GODOT_BIN=/opt/game-tools/godot/4.7.2-stable/Godot_v4.7.2-stable_linux.x86_64
export PG_BIN_DIR="$PWD/server/.local/pg/usr/lib/postgresql/18/bin"
"$GODOT_BIN" --headless --path prototypes/godot --script region_smoke.gd
EBASH_PILOT_MODE=construction-smoke bash server/check_godot_pilot.sh
```

Подставь скрипт или режим из нужной строки. Для настроек используй отдельный каталог данных в одном сеансе оболочки:

```bash
export XDG_DATA_HOME="$(mktemp -d)"
for arg in --write --verify; do
  "$GODOT_BIN" --headless --path prototypes/godot --script preferences_smoke.gd -- "$arg" || exit 1
done
```
