# Клиент EBash

## Текущая проверка

Рабочий клиент — **только игра** (D-192): Godot 4.7.2, Android arm64. Маршрут — [Launcher](godot/Launcher.gd) → [Region](godot/Region.gd) → карта [Main](godot/Main.gd) → предприятие [PilotPanel](godot/PilotPanel.gd) против сервера. Локальный UI-стенд без сервера не развивается. [Маршрут проверок](../docs/READING.md) · [HTTPS-мир](../docs/roadmap/p3-public-path.md). Последний кандидат на Lunex — `/home/gamedev/ebash-build/ebash-city-search-alpha-20261118.apk`. Подпись проверена. Ручной проход на телефоне открыт.

## Сборка

Установить Godot 4.7.2 и Android export templates 4.7.2, JDK и Android SDK, настроить пути в Editor Settings и debug keystore. Из корня: `godot --headless --path prototypes/godot --export-debug Android /путь/ebash-godot.apk`. Ключи подписи не хранить в репозитории.

## Проверка на телефоне

Одно устройство, landscape. Проверить запуск, читаемость, карту, компанию, пекарню, сетевые ошибки и повторный вход. Headless не заменяет проверку на Android.

Базовые автопроверки клиента: `godot --headless --path prototypes/godot --script smoke.gd` и `region_smoke.gd`. Серверные игровые сценарии — `EBASH_PILOT_MODE=… bash server/check_godot_pilot.sh` (имена `demo-smoke` / `profile-demo` — тестовые миры, не отдельный продукт).

## История

Каталог `unity/` и старые сравнительные заметки сохранены как архив выбора движка (D-092). Новые задачи по Unity или локальному «демозаводу» не ставятся.
