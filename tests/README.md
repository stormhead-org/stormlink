# Структура тестов StormLink

## Обзор

Этот каталог содержит все тесты для проекта StormLink. Тесты организованы по категориям и типам для удобства навигации и запуска.

## Структура каталогов

```
tests/
├── unit/                    # Юнит-тесты бизнес-логики
├── integration/             # Интеграционные тесты
├── performance/             # Тесты производительности
├── fixtures/                # Тестовые данные и вспомогательные функции
└── testcontainers/          # Настройка Docker контейнеров для тестов
```

## Типы тестов

### 1. Юнит-тесты (`tests/unit/`)
**Статус**: ⚠️ Частично работают (проблемы с foreign key constraints)

Тесты отдельных компонентов в изоляции:
- `comment_test.go` - Тестирование логики комментариев
- `post_test.go` - Тестирование логики постов

**Запуск**: 
```bash
go test ./tests/unit -v
```

### 2. Интеграционные тесты (`tests/integration/`)
**Статус**: ✅ Простые тесты работают / ❌ TestContainers требует Docker

- `user_test.go` - ✅ Простые интеграционные тесты (SQLite в памяти)
- `user_testcontainers_test.go` - ❌ Полные интеграционные тесты (PostgreSQL + Redis)

**Запуск простых тестов**:
```bash
go test ./tests/integration -run Simple -v
```

**Запуск полных тестов** (требует Docker):
```bash
go test ./tests/integration -v
```

### 3. Тесты производительности (`tests/performance/`)
**Статус**: ❌ Требует Docker и TestContainers

- `system_performance_test.go` - Тесты производительности системы

**Запуск** (требует Docker):
```bash
go test ./tests/performance -v
```

### 4. Usecase тесты (`server/usecase/*/`)
**Статус**: ✅ Работают / ⚠️ Частично работают

Тесты бизнес-логики на уровне use cases:
- `server/usecase/user/user_test.go` - ✅ Полностью работает
- `server/usecase/comment/comment_test.go` - ⚠️ Логические ошибки в тестах
- `server/usecase/community/community_test.go` - ⚠️ Проблемы с assertions
- `server/usecase/post/post_test.go` - ⚠️ Отсутствует поле slug

**Запуск конкретного usecase**:
```bash
go test ./server/usecase/user -v
go test ./server/usecase/comment -v
```

### 5. Service тесты (`services/*/internal/service/`)
**Статус**: ✅ Работают / ⚠️ Частично работают

Тесты сервисов:
- `services/auth/internal/service/service_test.go` - ✅ Полностью работает
- `services/mail/internal/service/service_test.go` - ⚠️ Проблемы с SMTP подключением
- `services/media/internal/service/service_test.go` - ⚠️ Nil pointer dereference

**Запуск**:
```bash
go test ./services/auth/internal/service -v
go test ./services/mail/internal/service -v
go test ./services/media/internal/service -v
```

## Тестовые данные и утилиты

### Fixtures (`tests/fixtures/`)
Содержит предопределенные тестовые данные и функции для их создания:
- `user.go` - Фикстуры для пользователей
- `extended.go` - Расширенные фикстуры для сложных сценариев

### TestContainers (`tests/testcontainers/`)
Настройка Docker контейнеров для интеграционных тестов:
- `setup.go` - Настройка PostgreSQL и Redis контейнеров

## Быстрый запуск

### Только работающие тесты
```bash
# Запустить все полностью работающие тесты
go test ./server/usecase/user ./services/auth/internal/service -v
go test ./tests/integration -run Simple -v
```

### Все тесты (включая проблемные)
```bash
go test ./... -v
```

### Конкретная категория
```bash
go test ./tests/unit -v          # Юнит-тесты
go test ./tests/integration -v   # Интеграционные тесты
go test ./server/usecase/... -v  # Все usecase тесты
go test ./services/.../service -v # Все service тесты
```

## Известные проблемы и их статус

### ✅ ИСПРАВЛЕНО
- **SQLite драйвер**: Все проблемы с `sql: unknown driver "sqlite3"` исправлены
- **Структура файлов**: Убраны дублирующиеся файлы и .disabled файлы
- **Нейминг**: Приведен к единообразному виду (service.go -> service_test.go)

### ⚠️ ЧАСТИЧНО РАБОТАЕТ
- **Comment тесты**: Проблемы с пагинацией и курсорами
- **Community тесты**: Неправильные assertions для указателей
- **Post тесты**: Отсутствует обязательное поле `slug`
- **Mail service**: Проблемы подключения к SMTP серверу
- **Media service**: Nil pointer dereference
- **Unit тесты**: Foreign key constraint violations

### ❌ НЕ РАБОТАЕТ
- **TestContainers тесты**: Требует настройки Docker окружения
- **Performance тесты**: Требует Docker и полную инфраструктуру

## Окружение для тестов

### Переменные окружения
```bash
export JWT_SECRET="test-jwt-secret-key-for-testing"
```

### Зависимости
- Go 1.24.2+
- SQLite (для простых тестов) - ✅ Работает
- Docker (для TestContainers тестов) - ❌ Требует настройки
- PostgreSQL (через Docker) - ❌ Требует настройки
- Redis (через Docker) - ❌ Требует настройки

## Рекомендации по разработке

### Написание новых тестов
1. **Юнит-тесты**: Размещайте рядом с тестируемым кодом (`package_test.go`)
2. **Интеграционные тесты**: Используйте `tests/integration/`
3. **Performance тесты**: Используйте `tests/performance/`

### Нейминг файлов
- Основной код: `service.go`
- Тесты: `service_test.go`
- НЕ используйте суффиксы типа `_simple_test.go`

### Изоляция тестов
```go
// Используйте уникальные базы данных для каждого теста
dbName := fmt.Sprintf("test_%s_%d", testName, time.Now().UnixNano())
client := enttest.Open(t, "sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared&_fk=1", dbName))
```

### Fixtures
```go
// Создавайте уникальные тестовые данные
testUser := fixtures.TestUser1
testUser.Email = fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
testUser.Slug = fmt.Sprintf("test-%d", time.Now().UnixNano())
```

## Следующие шаги

1. **Исправить логические ошибки** в частично работающих тестах
2. **Настроить Docker окружение** для TestContainers
3. **Добавить недостающие поля** в модели (например, `Post.slug`)
4. **Реализовать моки** для внешних зависимостей
5. **Настроить CI/CD** для автоматического запуска тестов

## Контакты и поддержка

При возникновении проблем с тестами:
1. Проверьте статус в этом файле
2. Убедитесь, что у вас установлены все зависимости
3. Проверьте переменные окружения
4. Для TestContainers убедитесь, что Docker запущен

**Основной статус**: Инфраструктурные проблемы решены, фокус на логике тестов.