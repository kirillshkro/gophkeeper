# tests-specs — `gophkeeper/internal/protocol` (контракт клиент–сервер)

## 1. Общие положения

- Инструменты: `go test -v -cover`, `encoding/json` (round-trip), `go.uber.org/mock` не нужен (пакет чистый).
- Пакет — контракт: клиент и сервер используют его для совместимости. Не зависит от реализации (`api`, `client`).
- Имена: `TestProtocol_<Payload>_<Сценарий>`, `TestProtocol_Validate<...>_...`.
- Ключевые ошибки: `ErrInvalidPayload`, `ErrUnsupportedVersion`, `ErrAuthRequired`.

## 2. Моки

| Зависимость | Заменитель |
|---|---|
| `data` (типы данных в payloads) | реальные (чистый пакет) |
| `encoding/json` | стандартная |
| — | моки не нужны |

## 3. Группы тестов

### 3.1. Версионирование протокола — `ProtocolVersion`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Текущая версия | валидный semver (regex `^\d+\.\d+\.\d+(-.*)?$`) |
| 2 | Соверстиместь (новый клиент ↔ старый сервер) | compatible version или `ErrUnsupportedVersion` |
| 3 | Поддерживаемая версия в запросе | принята, без ошибок |
| 4 | Незнательная/поддерживаемая версия | `ErrUnsupportedVersion` |

### 3.2. Payloads — валидация

#### `ValidateRegister(req *RegisterRequest) error`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Валидный `RegisterRequest` (логин, пароль, профиль) | `nil` |
| 2 | Не задан логин | `ErrInvalidPayload` (с указанием поля) |
| 3 | Не задан пароль | `ErrInvalidPayload` |
| 4 | Неправильный формат полей | `ErrInvalidPayload` |
| 5 | `nil`/пустой запрос | `ErrInvalidPayload` |

#### `ValidateLogin(req *LoginRequest) error`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Валидные credentials | `nil` |
| 2 | Не задан логин/пароль | `ErrInvalidPayload` |
| 3 | `nil` запрос | `ErrInvalidPayload` |

#### `ValidateData(req *DataRequest) error`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Валидный `DataRequest` (item + metadata) | `nil` |
| 2 | Неизвестный тип данных | `ErrUnknownType`/`ErrInvalidPayload` |
| 3 | Пустое содержимое (не Text) | `ErrEmptyData` |
| 4 | Устаревшая версия (update) | `ErrConflict`/`ErrInvalidPayload` |
| 5 | Не задан ownerID | `ErrInvalidPayload` |

#### `ValidateSync(req *SyncRequest) error`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Валидные изменения (item + action + version) | `nil` |
| 2 | Пустой список изменений | ok (no-op) |
| 3 | Неизвестное действие/версия | `ErrInvalidPayload` |
| 4 | `nil` запрос | `ErrInvalidPayload` |

### 3.3. Сериализация/десериализация (JSON round-trip)

| # | Payload | Ожидаемо |
|---|---|---|
| 1 | `RegisterRequest`/`RegisterResponse` | marshal → unmarshal — идентично |
| 2 | `LoginRequest`/`LoginResponse` | marshal → unmarshal — идентично |
| 3 | `DataRequest`/`DataResponse` | marshal → unmarshal — идентично |
| 4 | `SyncRequest`/`SyncResponse` | marshal → unmarshal — идентично |
| 5 | `Token` | marshal → unmarshal — идентично; в `Token` нет секретов (только ownerID, expiry, signature) |
| 6 | `Change` (item, action, version, timestamp) | marshal → unmarshal — идентично |
| 7 | Не валидный JSON | `ErrInvalidPayload` |
| 8 | JSON с неизвестными полями | загружен (extensibility), неизвестные поля проигнорированы |

### 3.4. Механизм аутентификации

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | `Authorization: Bearer <token>` | валидный формат |
| 2 | Формат токена (ownerID, expiry, signature) | корректный, без секретов |
| 3 | Заголовок отсутствует | `ErrAuthRequired` |
| 4 | Неверный формат заголовка (не Bearer) | `ErrAuthRequired` |

### 3.5. Совместимость (new client ↔ old server)

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Payload из новой версии на старом сервере | загрузился (extensibility) |
| 2 | Payload из старой версии на новом клиенте | загрузился (extensibility) |
| 3 | Несовместимая версия (major) | `ErrUnsupportedVersion` |

## 4. Дополнительные сценарии

- **Swagger/OpenAPI (необязательно):** если реализовано — валидация OpenAPI-схемы (генерация/проверка).
- **Контракт-тесты:** клиент и сервер используют одни и те же структуры — round-trip через `api`/`client` (интеграция).

## 5. Критерии готовности

- [ ] покрытие ≥ 70%;
- [ ] каждый payload + функция валидации покрыт;
- [ ] JSON round-trip для всех payloads — 100%;
- [ ] все ошибки (`ErrInvalidPayload`, `ErrUnsupportedVersion`, `ErrAuthRequired`) — через `errors.Is`;
- [ ] пакет не зависит от реализации (`api`, `client`) — проверено по импортам;
- [ ] godoc-документация у всех экспортированных элементов.
