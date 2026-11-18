# Маршруты чтения и проверок

Выбери одну строку. При известном ID сначала открой файл задачи и только релевантные ссылки из него. Полный список материалов есть в [индексе](INDEX.md).

| Работа | Прочитать сначала | Где менять и что проверять |
| --- | --- | --- |
| Демо и игровой интерфейс | [Запуск демо](../server/README.md#играбельная-демоверсия-пекарни), [карта Android](gdd/android-screen-flow.md) по теме | `prototypes/godot/PilotPanel.gd`, `PilotClient.gd`. Прогон `EBASH_PILOT_MODE=demo-smoke bash check_godot_pilot.sh` из `server/` |
| Сервер, тики, повтор команд | [Сервер](../server/README.md), [время](gdd/time.md) по заголовкам | `server/main.go`, `store.go`, `economy.go`. `go test ./...` и интеграция по README |
| Поставки, рынок, резерв | [Логистика](gdd/logistics.md), [рынок](gdd/market.md) по теме | `server/chain_supply.go`, `mill_trade.go`, `pilot_chains.go`. Сквозной прогон и `EBASH_TEST_MARKET_CONTENTION=1` по README |
| Один рецепт или цепочка | [DES-004](tasks/DES-004.md), [индекс каталога](catalogs/INDEX.md) | `python3 scripts/check_des004_catalog.py --chain R-BREAD-LOAF`. Подставь свой ID. Загружай нужные записи JSON |
| Цены и спрос | [DES-003](tasks/DES-003.md), [рынок](gdd/market.md), [правила цен](gdd/pricing-integrity.md) | `python3 scripts/check_des003_balance.py` и `python3 scripts/check_des003_examples.py` |
| Финансы и персонал | [DES-005](tasks/DES-005.md), [финансы](gdd/finance.md) или [персонал](gdd/workforce.md) | Меняй профильный раздел, запускай общий аудит |
| Город и выборы | [DES-006](tasks/DES-006.md), [развитие городов](gdd/city-development.md) или [география](gdd/geography.md) | Меняй профильный раздел, запускай общий аудит |
| APK и телефон | [Навык сборки Android](../.agents/skills/ebash-android-build/SKILL.md), [прототипы](../prototypes/README.md), [DES-007](tasks/DES-007.md) | Собери APK и проверь на устройстве по критериям сценария |

Решения ищи точечно в [реестре](decisions/001-foundation.md), например `grep -n 'D-125' docs/decisions/001-foundation.md`. Список разделов большого файла выдаёт `grep -n '^## ' файл`. Общий аудит запускается из корня через `python3 scripts/check_all.py`.
