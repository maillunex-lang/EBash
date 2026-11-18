# DES-004: матрица готовности рецептов к активации

Автоматически формируется скриптом `python3 scripts/report_des004_activation.py`.
Статус: **аудит, не разрешение исполнения**. Все 105 черновых операций остаются заблокированными.
Три утверждённых игровых профиля хранятся отдельно, не переводят исходные черновики в исполняемый статус.

## Сводка по отраслям

| Семейство | Черновики | Есть справочная запись | Отдельные игровые профили | Не хватает количества/единиц |
| --- | ---: | ---: | ---: | ---: |
| construction | 16 | 16 | 0 | 16 |
| dairy | 15 | 15 | 0 | 15 |
| glass | 12 | 12 | 0 | 12 |
| grain | 16 | 16 | 3 | 16 |
| oil | 14 | 14 | 0 | 14 |
| vegetables | 18 | 18 | 0 | 18 |
| wood | 14 | 14 | 0 | 14 |

## Ворота активации

1. Уточнить единицы и нормы каждого **складского** входа и выхода; участки и энергию не превращать в расходуемые товары.
2. Заполнить минимальную партию, оборудование/производительность, энергию, работников и качество с версией и происхождением нормы.
3. Согласовать совместные выходы и потери, особенно отруби, сыворотку и нефтяные фракции. Исключить выпуск бесплатных побочных продуктов.
4. Проверить сырьевые цепочки, ограничения 3×3, складскую доступность, резервы, себестоимость и торговые тики.
5. Подключать к исполняемому реестру только отдельно проверенный игровой профиль. Исторические черновики сохранять.

## Первая очередь: зерновая отрасль

| Рецепт | Версия | Игровой профиль | Незакрытые параметры | Складские количества |
| --- | ---: | --- | --- | --- |
| R-WHEAT-GRAIN | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | wheat_grain |
| R-WHEAT-FLOUR | 2 | отдельный утверждённый | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | wheat_grain, wheat_flour, wheat_bran |
| R-BREAD-DOUGH | 2 | отдельный утверждённый | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | bread_dough |
| R-BREAD-LOAF | 1 | отдельный утверждённый | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | bread_dough, bread_loaf |
| R-BREAD-PACK | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | bread_loaf, paper_pack, bread_pack |
| R-BARLEY-GRAIN | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | barley_grain |
| R-BARLEY-FLOUR | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | barley_grain, barley_flour |
| R-RYE-GRAIN | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | rye_grain |
| R-RYE-FLOUR | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | rye_grain, rye_flour |
| R-RYE-DOUGH | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | rye_flour, water, yeast, rye_dough |
| R-RYE-BREAD | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | rye_dough, rye_bread |
| R-OAT-GRAIN | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | oat_grain |
| R-OAT-FLAKES | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | oat_grain, oat_flakes |
| R-ANIMAL-FEED | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | wheat_bran, barley_grain, animal_feed |
| R-PASTA-DOUGH | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | wheat_flour, water, pasta_dough |
| R-DRY-PASTA | 1 | отсутствует | min_batch, equipment_profile, energy_profile, workforce_profile, quality_profile | pasta_dough, dry_pasta |

## Ограничения текущего аудита

Наличие справочной записи не означает подтверждения технологических норм. Полная проверка материального баланса всех 105 операций невозможна до заполнения количеств. Три отдельных хлебных игровых профиля не покрывают весь зерновой каталог.
