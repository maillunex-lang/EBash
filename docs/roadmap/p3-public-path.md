# P3 — публичный HTTPS-маршрут (18.11.2026)

Согласованный пользователем путь: `https://rikagro.ru/games/bash/`.

- `GET /games/bash/`: отдельная статическая страница EBash, HTTP 200.
- `GET /games/bash/ebash-test.apk`: Android arm64 debug APK, HTTP 200, подпись проверена. Публикуется из `/var/www/ebash-p3/ebash-test.apk`.
- `/games/bash/api/v1/worlds/ebash-p3-test/...`: nginx proxy на `127.0.0.1:18749`, с сохранением заголовка Authorization. Без токена 401, с токеном 200. API и PostgreSQL по-прежнему изолированы, PostgreSQL не имеет TCP listener.
- Конфигурация nginx прошла `nginx -t`, выполнен reload. Резервная копия исходной конфигурации `/etc/nginx/rikagro-game.ebash-backup-20261118` (не в `sites-enabled`). Остальные пути не переназначены.
- Для входа в Android: HTTPS-адрес `https://rikagro.ru/games/bash/api`, мир `ebash-p3-test`, индивидуальный токен выдаётся отдельно; **не публиковать токен на странице, в APK или документах**.
- Публичный API доступен в режиме тестового мира, не production. Отсутствует проверка интерфейса на реальном Android; она согласована на P4. Не считать APK финальным релизом.

Перед более широким тестированием добавить rate limiting, отдельные тестовые учётные записи и контролируемый сброс мира. На P3 не изменять производственные данные других проектов.
