# tests-specs — `gophkeeper/cmd/client` (точка входа CLI)

## 1. Общие положения

- Инструменты: `go test -v -cover`, `go.uber.org/mock`, захват stdout/stderr.
- Пакет `main` — точка входа CLI: сборка компонентов, запуск CLI. Не содержит бизнес-логики.
- Имена: `TestMain_Run_...`, `TestMain_ParseFlags_...`.

## 2. Моки

| Зависимость | Заменитель |
|---|---|
| `config` (`LoadClientConfig`) | gomock / реальные файлы (`t.TempDir()`) |
| `client.Client` (`NewClient`) | gomock |
| `cli` (`Run`) | gomock / реальные (с mock'ами) |
| `version` | реальные |

## 3. Группы тестов

### 3.1. Точка входа — `main()` / `run() error`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Успешный запуск (все компоненты ок) | exit 0 |
| 2 | Ошибка загрузки конфигурации | exit ≠ 0, `ErrConfigLoadFailed` (`errors.Is`) |
| 3 | Ошибка инициализации клиента | exit ≠ 0, `ErrClientInitFailed` |
| 4 | Ошибка запуска CLI | exit ≠ 0, `ErrCLIStartFailed` |
| 5 | Флаг `version`/`-v` | вывод версии и даты сборки, exit 0 |
| 6 | Флаг `help`/`-h` | вывод справки, exit 0 |

### 3.2. Парсинг флагов — `parseFlags()`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Валидные флаги (`-server`, `--help`, `-v`) | корректные `cli.Options` |
| 2 | Неизвестный флаг | ошибка usage, exit ≠ 0 |
| 3 | Пустой `os.Args` | error usage |

### 3.3. Сборка (SOLID)

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Нет бизнес-логики | статическая проверка (review): только `config`/`client`/`cli`/`version` |
| 2 | Вывод только через `slog` | без `fmt` |
| 3 | Ошибки не пробрасываются в stdout | stderr + exit code |

## 4. Дополнительные сценарии

- **Кроссплатформенность:** кросс-компиляция CLI (Windows, Linux, macOS) + запуск `version` (интеграционный smoke).
- **Exit codes:** `0` — успех, `≠0` — ошибка (по SPECIFICATION.md).
- **Graceful failure:** при ошибке — понятное сообщенье в stderr, exit code ≠ 0.

## 5. Критерии готовности

- [ ] покрытие ≥ 70%;
- [ ] все ошибки (`ErrConfigLoadFailed`, `ErrClientInitFailed`, `ErrCLIStartFailed`) — через `errors.Is`;
- [ ] коды выхода: `0` на успехе, `≠0` на ошибке;
- [ ] кросс-платформенная сборка и запуск (`GOOS=linux/windows/darwin`);
- [ ] godoc-документация.
