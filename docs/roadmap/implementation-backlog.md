# Пакеты реализации EBash

Следующий клиентский пакет — [E-106](#e-106). Для одной задачи достаточно её раздела. [Этапы](stages.md) · [команды проверок](../READING.md) · [журнал испытаний](implementation-history.md).

## E-101

Тесты атомарности и запрета двойного выпуска, пустые/отрицательные ресурсы, ошибочные профили и overflow

- Открыть [game_profile_execution.go](../../server/game_profile_execution.go), [game_profile_execution_test.go](../../server/game_profile_execution_test.go).
- Приёмка — Нет отрицательных остатков/двойного выпуска, входы не мутируются при отказе.
- Проверка из `server/` — `go test -race ./...`.

## E-102

Полное DES-002 планирование по связанным ресурсам, заявкам и замороженным квотам

- Открыть [game_profile_optimizer.go](../../server/game_profile_optimizer.go), [game_profile_stages.go](../../server/game_profile_stages.go), [DES-002.md](../../docs/tasks/DES-002.md).
- Приёмка — Лексикографическая оптимальность малых групп; корректная квота и перераспределение; 9+ безопасный fallback.
- Проверка из `server/` — `go test -run "TestGame(Optimizer|Profile)" ./...`.

## E-103

Проверить циклы и общие мощности 3×3, качество смешивания и совместимость

- Открыть [game_profile_stages.go](../../server/game_profile_stages.go), [production.md](../../docs/gdd/production.md), [facility-grid.md](../../docs/gdd/facility-grid.md).
- Приёмка — Одна линия один раз за тик, без бесплатных промежуточных товаров и конфликтов ресурсов.
- Проверка из `server/` — `go test -run TestGameProfile ./...`.

## E-104

Расширить проверенную хлебную интеграцию на общие склады и торговлю. Включение в штатный тик зависит от критических проверок E-101–103.

- Открыть [game_profile_trade.go](../../server/game_profile_trade.go), [game_profile_inventory_bridge.go](../../server/game_profile_inventory_bridge.go), [game_profile_tick.go](../../server/game_profile_tick.go).
- Приёмка — Поставка T→производство T+1, выпуск T→продажа T+1, атомарные HTTP/PostgreSQL тесты.
- Проверка из корня — `bash server/check_p32_postgres.sh`.

## E-105

Реальные игровые профили DES-004 и 3×3 по отраслевым пакетам

- Открыть [INDEX.md](../../docs/catalogs/INDEX.md), [DES-004.md](../../docs/tasks/DES-004.md).
- Приёмка — Схема, вход/выход, источники, безопасная активация без `draft_not_executable`.
- Проверка — `python3 scripts/check_all.py`.

## E-106

Развивать общий интерфейс компании. Ближайший шаг — экран предприятий с переходом от состояния компании к своему участку, стройке или действующему объекту. Цены и наличие берутся с сервера.

- Открыть [Region.gd](../../prototypes/godot/Region.gd), [Main.gd](../../prototypes/godot/Main.gd), [PilotPanel.gd](../../prototypes/godot/PilotPanel.gd), [android-screen-flow.md](../../docs/gdd/android-screen-flow.md).
- Приёмка — APK, ручной Android-проход, адаптация под вырезы и ландшафт, ошибки сети.
- Первый пакет готов, когда из «Моя компания» можно открыть площадку до стройки, увидеть срок во время стройки и перейти к управлению после приёмки.
- Проверка — `region_smoke.gd` и `construction-smoke` по [командам Lunex](../READING.md#запуск-клиентских-проверок-на-lunex). Для закрытия всего E-106 нужен ручной проход на Android.

## E-107

Безопасность и надёжность MMO: конкурентные транзакции, рестарт, тик-догонялка, авторизация

- Открыть [store.go](../../server/store.go), [integration_test.go](../../server/integration_test.go), [DES-001.md](../../docs/tasks/DES-001.md).
- Приёмка — Изолированные тесты, журнал без дублей, восстановление после отказов.
- Проверка из корня — `PG_BIN_DIR="$PWD/server/.local/pg/usr/lib/postgresql/18/bin" bash server/check_restore.sh`.

## E-108

Альфа-баланс, нагрузка и долгосрочная экономика

- Открыть [alpha-balance-lab.md](../../docs/roadmap/alpha-balance-lab.md).
- Приёмка — Воспроизводимые 12/36/120 месяцев, отчёты о прибыли, ценах, дефиците, p95.
- Проверка — записать результаты для каждого горизонта по [матрице сценариев](alpha-balance-lab.md#базовая-матрица-сценариев).
