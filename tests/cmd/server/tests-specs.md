# tests-specs — `gophkeeper/cmd/server` (точка входа сервера)

## 1. Общие положения

- Инструменты: `go test -v -race -cover`, `go.uber.org/mock`, `net/http/httptest`, инъекция сигналов (`os/signal`).
- Пакет `main` — точка входа сервера: сборка компонентов, жизненный цикл (старт, graceful shutdown). Не содержит бизнес-логики.
- Имена: `TestServer_Run_...`, `TestServer_SetupServer_...`, `TestServer_Shutdown_...`.

## 2. Моки

| Зависимость | Заменитель |
|---|---|
| `config` (`LoadServerConfig`) | gomock / реальные файлы (`t.TempDir()`) |
| `storage` (`NewStorage`) | gomock / in-memory SQLite (GORM Gen) |
| `api.Server` (`NewServer`, `Serve`) | gomock / реальный (с mock-ами ниже) |
| Сигналы (`SIGTERM`/`SIGINT`) | инъекция (mock signal channel / `os/signal` с заменой) |

## 3. Группы тестов

### 3.1. Точка входа — `main()` / `run(cfg *config.Config) error`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Валидная конфигурация | сервер запущен, exit 0 (после остановки) |
| 2 | Ошибка загрузки конфигурации | exit ≠ 0, `ErrConfigLoadFailed` |
| 3 | Ошибка инициализации хранилища | exit ≠ 0, `ErrServerStartFailed` |
| 4 | Ошибка старта HTTP-сервера (порт занят) | exit ≠ 0, `ErrServerStartFailed` |
| 5 | Сигнал `SIGTERM` / `SIGINT` | graceful shutdown, exit 0 |
| 6 | Сигнал в момент старта | корректная обработка, exit ≠ 0/0 (по дизайну) |

### 3.2. Сборка сервера — `setupServer(...) *http.Server`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Все маршруты из `protocol` | зарегистрированы |
| 2 | TLS конфигурация (если задана) | TLS-сервер (минимальная версия TLS1.2) |
| 3 | Middleware (`auth`/`authz`) | подключены ко всем защищённым эндпоинтам |
| 4 | `nil`/пустая конфигурация | ошибка инициализации |

### 3.3. Graceful shutdown

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | `SIGTERM` во время активного запроса | активный запрос завершён, затем стоп, exit 0 |
| 2 | `SIGINT` | аналогично `SIGTERM` |
| 3 | Shutdown без активных запросов | быстрая остановка, exit 0 |
| 4 | Ошибка shutdown (e.g. порт занят) | `ErrShutdownFailed`, exit ≠ 0 |
| 5 | Повторный shutdown (double) | без panic, idempotent |

### 3.4. Логирование

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Старт сервера | `slog`-лог (адрес, порт, TLS on/off) |
| 2 | Остановка сервера | `slog`-лог (причины, время работы) |
| 3 | Ошибки | `slog`-лог (без раскрытия секретов) |
| 4 | Секреты (пароли, токены) в логах | отсутствуют (захват handler) |

### 3.5. Интеграционный smoke

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | `go test -v -race ./cmd/server` | все тесты зелёные |
| 2 | Запуск сервера с реальной конфигурацией (в тесте) | HTTP-эндпоинты работают (httptest) |

## 4. Дополнительные сценарии

- **Graceful shutdown:** тесты с реальными запросами во время shutdown (не только mock'и).
- **SOLID:** нет бизнес-логики в `main`/`run` — только композиция и жизненный цикл (статическая проверка).
- **Кроссплатформенность:** сборка сервера для Windows, Linux, macOS (интеграционный smoke).

## 5. Критерии готовности

- [ ] покрытие ≥ 70%;
- [ ] жизненный цикл (старт, стоп, shutdown) — полностью покрыт;
- [ ] все ошибки (`ErrConfigLoadFailed`, `ErrServerStartFailed`, `ErrShutdownFailed`) — через `errors.Is`;
- [ ] graceful shutdown проверен (включая активные запросы);
- [ ] кросс-платформенная сборка (GOOS=linux/windows/darwin);
- [ ] godoc-документация;
- [ ] логирование через `slog`, без `fmt`;
- [ ] секреты не в логах.
