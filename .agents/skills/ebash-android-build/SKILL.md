---
name: ebash-android-build
description: Собрать Android APK клиента EBash из Godot, проверить подпись и подготовить проверку на телефоне. Использовать, когда просят APK, сборку, экспорт или замеры на устройстве.
---

# Сборка Android

Команды и текущее состояние — в [README прототипов](../../../prototypes/README.md); здесь только порядок действий.

1. Нужны: Godot 4.7.2 (D-094) и Android export templates той же версии, JDK, Android SDK. Пути — в Editor Settings.
2. Пресет `Android` в `prototypes/godot/export_presets.cfg`: только arm64-v8a. Пути к keystore и пароли в репозиторий не коммить: задавай локально или через переменные окружения Godot (`GODOT_ANDROID_KEYSTORE_DEBUG_PATH` и др.).
3. Сначала smoke: `godot --headless --path prototypes/godot --script smoke.gd`.
4. Экспорт: `godot --headless --path prototypes/godot --export-debug Android <путь>/ebash-godot.apk`. APK кладётся вне репозитория (`*.apk` в `.gitignore`).
5. Проверь подпись: `apksigner verify --verbose <apk>`.
6. На телефоне: запуск, читаемость, жесты, переходы, FPS при 30/60, нагрев и память после одинакового времени. Устройство и результаты записать в «Текущая проверка» README прототипов.

Не сообщай «APK проверен», если он только собран: различай «собран», «подпись проверена», «проверен на устройстве».
