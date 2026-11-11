# internal/api

## Назначение

Пакет `api` реализует HTTP-интерфейс сервера GophKeeper. Это слой
представления: HTTP-хендлеры, маршруты, обработка запросов/ответов.

Пакет не содержит бизнес-логики — он делегирует работу в сервисы
(`user`, `sync`, `data`, `storage`). Это реализация SOLID и Separation of
Concerns.

## Отвечает за

- HTTP-хендлеры и маршруты;
- регистрацию и аутентификацию (через `user`/`auth`);
- CRUD данных (через `data`/`storage`/`sync`);
- синхронизацию (через `sync`);
- передачу приватных данных владельцу по запросу;
- обработку ошибок и форматирование HTTP-ответов.

## Требования

### Функциональные

1. Реализовать эндпоинты (по протоколу из `protocol`):
   - `POST /register` — регистрация;
   - `POST /auth/login` — аутентификация;
   - `GET /data` — получение данных;
   - `POST /data` — создание;
   - `PUT /data/:id` — редактирование;
   - `DELETE /data/:id` — удаление;
   - `POST /sync` — синхронизация;
   - `GET /data/:id` — передача данных владельцу по запросу.
2. Интегрировать middleware аутентификации (`auth`) и авторизации (`authz`)
   в хендлеры — защита всех защищённых эндпоинтов.
3. Обработать ошибки бизнес-логики в HTTP-статусы:
   - 400 (невалидные данные), 401 (нет аутентификации), 403 (нет доступа),
     404 (не найдено), 409 (конфликт), 500 (ошибка сервера).
4. Вернуть JSON (или формат из `protocol`) для ответов.
5. Логировать запросы/ошибки через `slog` (без раскрытия чувствительных
   данных).

### Неразметочные

- Хендлеры не должны содержать бизнес-логики (SOLID).
- `(value, error)`; godoc; тесты >= 70%.
- Не более 250 строк на файл.
- Вывод через `slog`, не `fmt`.
- Методы/хендлеры принимают не более двух параметров.
- Сервер должен поддерживать graceful shutdown (перенесено в `cmd/server`).

## Ожидаемое API

- `NewServer(cfg *config.Config, ...) *Server` — композиция хендлеров.
- `type Server struct{...}` с методом `Serve() error`.
- Хендлеры: `HandleRegister`, `HandleLogin`, `HandleData` (CRUD),
  `HandleSync`, `HandleFetch`.
- Использование `net/http` + `encoding/json` (или формат из `protocol`).
- Middleware: `AuthMiddleware`, `AuthzMw`.

## Зависимости

- `gophkeeper/internal/protocol` (контракт эндпоинтов/payloads).
- `gophkeeper/internal/auth`, `gophkeeper/internal/authz`.
- `gophkeeper/internal/user`, `gophkeeper/internal/sync`,
  `gophkeeper/internal/data`, `gophkeeper/internal/storage`,
  `gophkeeper/internal/config`.
- Стандартная библиотека `net/http`.
