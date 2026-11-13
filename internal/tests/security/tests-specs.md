# tests-specs — `gophkeeper/internal/security` (криптографическое ядро)

## 1. Общие положения

- Инструменты: `go test -v -cover`, захват `slog.Handler` для проверки логов.
- Пакет чистый: не зависит от других пакетов модуля; только stdlib + `golang.org/x/crypto` (при необходимости).
- Имена: `TestSecurity_<Функция>_<Сценарий>`, например `TestSecurity_EncryptDecrypt_RoundTrip`.
- **Важно:** перед реализацией проверить актуальную документацию (context7/fetch) для алгоритмов (argon2/bcrypt/pbkdf2, AES-GCM, HMAC).

## 2. Моки

| Зависимость | Заменитель |
|---|---|
| — | Пакет не имеет зависимостей; моки не нужны |
| Файловая система | не нужна (чистая криптография) |

## 3. Группы тестов

### 3.1. Хэширование паролей — `HashPassword(plain string) (string, error)`, `VerifyPassword(plain, hash string) error`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | `HashPassword(валидный пароль)` | валидный хэш (формат: salt+hash), `nil` |
| 2 | `VerifyPassword(верный, хэш)` | `nil` |
| 3 | `VerifyPassword(неверный пароль, хэш)` | `ErrInvalidPasswordHash`/non-nil |
| 4 | `VerifyPassword(верный пароль, повреждённый хэш)` | non-nil |
| 5 | `HashPassword("")` | ошибка (`ErrEmptyData`/`ErrInvalidPassword`) |
| 6 | Разные plaintext → разные хэши (salt) | детерминированность (одинаковый пароль + одинаковый salt → одинаковый хэш; разный salt → разный хэш) |
| 7 | **Секреты в хэше/логe** | пароль в открытом виде отсутствует в хэше и в `slog`-записях |

### 3.2. Шифрование/дешифрование — `Encrypt(plain, key) ([]byte, error)`, `Decrypt(cipher, key) ([]byte, error)`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | **Round-trip для всех типов данных** (text, binary, credential, card) | содержимое идентично |
| 2 | Шифрование с неверным ключом | `ErrDecryptionFailed` |
| 3 | Повреждённый ciphertext (AEGS) | `ErrDecryptionFailed` |
| 4 | Пустой plaintext | корректное поведение (зафиксировано: ok/ошибка) |
| 5 | Очень крупный payload (MB) | round-trip без ошибок |
| 6 | **Секреты в ciphertext/логe** | plaintext отсутствует в ciphertext; ключ не в `slog` |

### 3.3. Управление ключами — `GenerateMasterKey() ([]byte, error)`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | `GenerateMasterKey()` | длина по спецификации (e.g. 32 bytes), высокая энтропия |
| 2 | Два вызова → разные ключи | true |
| 3 | **Ключ в логах** | не в `slog` (захват handler) |
| 4 | Повторное использование ключа (сafety) | при неправильном использовании — ошибка |

### 3.4. Подпись и проверка токенов — `SignToken(payload, key) ([]byte, error)`, `VerifyToken(signature, payload, key) error`

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | `SignToken` → `VerifyToken` (round-trip) | `nil` |
| 2 | Проверка с неверным ключом | `ErrInvalidSignature` |
| 3 | Проверка после повреждения payload | `ErrInvalidSignature` |
| 4 | Проверка истёкшего токена (если TTL в payload) | ошибка (не `ErrTokenExpired`, а `ErrInvalidSignature`/время) |
| 5 | `nil`/пустые входные данные | соответствующая ошибка (`ErrInvalidSignature`/`ErrKeyInvalid`) |

### 3.5. Безопасная передача (TLS, server-side)

| # | Сценарий | Ожидаемо |
|---|---|---|
| 1 | Конфигурация TLS (minVersion TLS1.2, разрешённые cipher suites) | небезопасные шифры/протоколы отключены |
| 2 | Сертификат/ключ — генерация/валидация (если реализовано) | корректная TLS-соединение (round-trip) |

## 4. Дополнительные сценарии

- **Актуальность алгоритмов:** перед внедрением — актуальная документация (context7) по argon2/bcrypt/pbkdf2, AES-GCM, HMAC.
- **Edge cases:** пустые/нулевые входные данные, очень большие payloads, Unicode (UTF-8).
- **Паники:** не возникают при битых данных/nil/пустых ключах.
- **Секреты:** ни ключи, ни пароли, ни данные карт не попадают в `slog`, ответы, panic-сообщения.

## 5. Критерии готовности

- [ ] покрытие ≥ 70% (включая round-trip шифрования и edge-cases);
- [ ] все ошибки (`ErrInvalidPasswordHash`, `ErrDecryptionFailed`, `ErrInvalidSignature`, `ErrKeyInvalid`) — через `errors.Is`;
- [ ] godoc-документация у всех экспортированных элементов;
- [ ] секреты не в логах (тесты-ассерты по захваченному `slog.Handler`);
- [ ] пакет независим (не зависит от `data`, `user`).
