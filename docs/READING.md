# Быстрый выбор документов

При передаче проекта один раз прочитать [HANDOFF](HANDOFF.md). Для общего обзора правил без большого GDD открыть [краткие инварианты](gdd/QUICK-REFERENCE.md). Для известного ID задачи открыть её файл напрямую. Для выбора работы — [очередь](tasks/INDEX.md). Полный GDD для каждой задачи читать не требуется.

| Работа | Начать с | Затем при необходимости |
| --- | --- | --- |
| Понять готовность проекта | [Статус](STATUS.md) | [Этапы](roadmap/stages.md) |
| Серверный расчёт | [Время](gdd/time.md) | [DES-001](tasks/DES-001.md) |
| Завод и рецепты | [DES-004](tasks/DES-004.md), [каталог](catalogs/INDEX.md) | [Производство](gdd/production.md) по нужным разделам, [сетка 3×3](gdd/facility-grid.md) |
| Финансы и персонал | [DES-005](tasks/DES-005.md), [финансирование](gdd/finance.md) | [Персонал](gdd/workforce.md), [логистика](gdd/logistics.md) |
| Android UX | [DES-007](tasks/DES-007.md), [интерфейс](gdd/ui.md) | [Карты](gdd/maps.md), [обучение](gdd/onboarding.md) |
| Торговля и баланс | [Рынок](gdd/market.md) | [Городское снабжение](gdd/city-supply.md), [защита средств](gdd/pricing-integrity.md) |
| Работа без игрока | [Управляющие](gdd/managers.md) | [Монетизация](gdd/monetization.md) |
| Город и выборы | [Развитие городов](gdd/city-development.md) | [География](gdd/geography.md) |

Поиск из корня: `grep -n 'D-042' docs/decisions/001-foundation.md`; `grep -n '^## ' docs/gdd/market.md`; `grep -n 'Статус:' docs/tasks/DES-*.md`. Для списка файлов использовать `find docs -type f`.

Профильный GDD хранит подробные правила и предложения. Реестр решений — краткий журнал выбранного со ссылками. Очередь хранит порядок задач; файл задачи — её входы и приёмку. Не копировать формулы в задачи и журналы.
