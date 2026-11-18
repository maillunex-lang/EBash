# Маршруты чтения для ИИ-исполнителя

**Цель: не читать весь репозиторий.** Если известен ID E-101…E-108, открой только строку [backlog](roadmap/implementation-backlog.md) и перечисленные там файлы. Для проектной задачи DES-XXX открой **один** `docs/tasks/DES-XXX.md`. Реестр решений читать точечно по `D-XXX` через `grep -n`. Не загружай `docs/decisions/001-foundation.md`, `docs/tasks/DES-004-history.md` или все JSON целиком без необходимости.

| Тип задачи | Первые файлы | Минимальная проверка |
| --- | --- | --- |
| Тики, атомарность, API | `server/store.go`, `server/integration_test.go`, нужный Go-модуль | `cd server && go test ./...` |
| Оптимизатор/производство | `server/game_profile_execution.go`, `game_profile_optimizer.go`, `game_profile_stages.go`; [E-101–103](roadmap/implementation-backlog.md) | `cd server && go test -run TestGameProfile ./...` |
| Поставки, рынок, резервы | `server/chain_supply.go`, `mill_trade.go`, `game_profile_tick.go`; [E-104](roadmap/implementation-backlog.md) | `cd server && go test ./...`, при необходимости `bash check_p32_postgres.sh` |
| Рецепт или отрасль | [DES-004](tasks/DES-004.md), [индекс каталога](catalogs/INDEX.md); только нужный ID | `python3 scripts/check_des004_catalog.py --chain R-BREAD-LOAF` (заменить ID) |
| DES-003, цены и спрос | [DES-003](tasks/DES-003.md), `docs/gdd/market.md` | `python3 scripts/check_des003_balance.py` |
| Энциклопедия лендинга | `web/landing/index.html`, `web/wiki/index.html`, `web/wiki/wiki.js`, `scripts/build_wiki_catalog.py` | `python3 scripts/check_wiki.py`; Nginx `/games/bash/wiki/` |
| Android/Godot | `prototypes/godot`, [маршрут экранов](gdd/android-screen-flow.md), [навык](../.agents/skills/ebash-android-build/SKILL.md) | `cd server && EBASH_PILOT_MODE=demo-smoke bash check_godot_pilot.sh`; ручной Android |
| Финансы/город/архитектура | Только соответствующий `docs/tasks/DES-005.md`, `DES-006.md` или `DES-008.md` | Профильные проверки и `python3 scripts/check_all.py` |

**Перед работой:** `git status --short`, [AGENTS](../AGENTS.md). **После:** целевые тесты, `python3 scripts/check_all.py`, `git diff --check`; обновить только профильный протокол и краткий [STATUS](STATUS.md) при изменении готовности. [Аудит рисков](audits/2026-11-18.md) читать при работе с E-101–104, не при каждой UI-задаче.

## Рецепты, мебель, магазины и аудит без лишнего чтения

| Запрос | Открыть только | Проверить |
| --- | --- | --- |
| Мебельный завод | `docs/catalogs/furniture.proposal.json` | `python3 scripts/check_des004_furniture.py` |
| Магазины и розничные SKU | `docs/catalogs/retail-channels.proposal.json` | `python3 scripts/check_des004_retail_channels.py` |
| Зерновые сценарии | `docs/catalogs/grain-scenarios.proposal.json` | `python3 scripts/check_des004_grain_scenarios.py` |
| Экономика зерна | `docs/catalogs/grain-economy.audit.json` | `python3 scripts/check_des004_grain_economy.py` |
| Аудит всего дерева исходников | `scripts/audit_project_inventory.py`, `docs/audits/2026-11-18-full-project-followup.md` | `python3 scripts/check_all.py` |

**Не читать без необходимости:** `server/.local` (тестовые БД и артефакты), большие исторические GDD, `docs/tasks/DES-004-history.md`, `.balance-tools-venv`. Новый исполнитель не должен считать неисполняемые предложения действующими рецептами.
