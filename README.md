# EBash

Рабочее обозначение Android-игры: реалистичный экономический онлайн-симулятор развития от небольшого бизнеса до корпорации. Текущее рабочее название: «БАШБАБКИ».

ИИ-исполнителям: для передачи контекста открыть [HANDOFF](docs/HANDOFF.md), для правил [AGENTS.md](AGENTS.md) (для Claude — [CLAUDE.md](CLAUDE.md), для Copilot — [инструкции](.github/copilot-instructions.md)). Навыки: [задача](.agents/skills/ebash-task/SKILL.md), [решение владельца](.agents/skills/ebash-decision/SKILL.md), [Godot](.agents/skills/ebash-godot/SKILL.md), [сборка APK](.agents/skills/ebash-android-build/SKILL.md), [прототипы](.agents/skills/ebash-prototype/SKILL.md).

[Текущее состояние](docs/STATUS.md) · [дорожная карта](docs/roadmap/stages.md) · [запуск демо](server/README.md#играбельная-демоверсия-пекарни) · [карта документации](docs/INDEX.md).

Единая офлайн-проверка из корня: `python3 scripts/check_all.py`. [Краткие утверждённые правила](docs/gdd/QUICK-REFERENCE.md).

**GitHub не обновлять до отдельного распоряжения владельца.** Исходная рабочая копия находится на VPS Lunex в `/home/gamedev/ebash-main`; [текущий статус](docs/STATUS.md) является единой сводкой, а не отметкой готовности игры.

**Сайт БАШБАБКИ:** [лендинг](https://rikagro.ru/games/bash/) · [энциклопедия товаров и рецептов](https://rikagro.ru/games/bash/wiki/). Исходники энциклопедии: `web/wiki/index.html`; JSON создаётся командой `python3 scripts/build_wiki_catalog.py` из утверждённых профилей и черновых каталогов DES-004. Черновики не становятся исполняемыми при публикации. На VPS сайт находится отдельно в `/var/www/ebash-p3`, Nginx-маршрут `/games/bash/wiki/`.

Исходник главной страницы сайта сохранён в `web/landing/index.html` (копия размещённого лендинга и его статических ассетов); исходники энциклопедии в `web/wiki/`. Изменения на VPS не публикуются на GitHub автоматически.
