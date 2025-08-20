# 🔐 Анализ архитектуры аутентификации и авторизации

## 📋 Обзор архитектуры

Проект использует **микросервисную архитектуру** с разделением ответственности между сервисами:

### 🏗️ Структура сервисов

```
stormlink/
├── server/           # GraphQL API Gateway
├── services/
│   ├── auth/         # Сервис аутентификации
│   ├── user/         # Сервис пользователей
│   ├── mail/         # Сервис почты
│   └── media/        # Сервис медиа
└── shared/           # Общие библиотеки
    ├── auth/         # Контекст аутентификации
    ├── jwt/          # JWT утилиты
    └── http/         # HTTP утилиты
```

## 🔐 Аутентификация

### 1. **JWT Токены**

**Типы токенов:**
- **Access Token** (15 минут) - для API запросов
- **Refresh Token** (7 дней) - для обновления access токена

**Структура токенов:**
```json
{
  "user_id": "1",
  "exp": 1734567890,
  "type": "access|refresh"
}
```

### 2. **Поток аутентификации**

```
1. Login Request → Auth Service
2. Validate Credentials → Database
3. Generate JWT Tokens → Shared/JWT
4. Set Cookies → Shared/HTTP
5. Return Tokens → Client
```

### 3. **Middleware аутентификации**

**HTTPAuthMiddleware** (`server/middleware/http_auth.go`):
- Извлекает токен из `Authorization` заголовка или куки `auth_token`
- Валидирует токен через gRPC вызов к Auth Service
- Устанавливает `userID` в контекст через `sharedauth.WithUserID()`

### 4. **Auth Service** (`services/auth/`)

**Основные методы:**
- `Login()` - аутентификация пользователя
- `ValidateToken()` - валидация access токена
- `RefreshToken()` - обновление токенов
- `GetMe()` - получение текущего пользователя
- `Logout()` - выход из системы

**Особенности:**
- ✅ Поддержка Redis для revoke токенов
- ✅ Автоматическая ротация refresh токенов
- ✅ Установка HTTP куки
- ✅ Валидация email перед входом

## 🛡️ Авторизация

### 1. **Система ролей**

**Уровни ролей:**
- **Host Roles** - роли на уровне платформы
- **Community Roles** - роли в сообществах

**Права ролей:**
```go
// Host Roles
- communityRolesManagement
- hostUserBan
- hostUserMute
- hostCommunityPostDelete
- hostCommunityCommentsDelete

// Community Roles  
- communityRolesManagement
- communityUserBan
- communityUserMute
- communityDeletePost
- communityDeleteComments
```

### 2. **Проверка прав**

**В резолверах:**
```go
// Проверка владельца сообщества
if cm.OwnerID != currentUserID {
    permsMap, err := r.UserUC.GetPermissionsByCommunities(ctx, currentUserID, []int{cid})
    perms := permsMap[cid]
    if perms == nil || !perms.CommunityRolesManagement {
        return nil, fmt.Errorf("forbidden")
    }
}
```

### 3. **Система банов и мутов**

**Типы ограничений:**
- `HostUserBan` - бан пользователя на уровне платформы
- `HostCommunityBan` - бан сообщества на уровне платформы
- `CommunityUserBan` - бан пользователя в сообществе
- `CommunityUserMute` - мут пользователя в сообществе

## 🔧 Техническая реализация

### 1. **Контекст аутентификации** (`shared/auth/`)

```go
// Типобезопасный ключ контекста
type typedKey struct{ name string }
var userIDKey = typedKey{name: "userID"}

// Установка userID в контекст
func WithUserID(ctx context.Context, id int) context.Context

// Извлечение userID из контекста
func UserIDFromContext(ctx context.Context) (int, error)
```

### 2. **JWT утилиты** (`shared/jwt/`)

```go
// Генерация токенов
func GenerateAccessToken(userID int) (string, error)
func GenerateRefreshToken(userID int) (string, error)

// Парсинг токенов
func ParseAccessToken(tokenString string) (*AccessTokenClaims, error)
func ParseRefreshToken(tokenString string) (*RefreshTokenClaims, error)
```

### 3. **HTTP утилиты** (`shared/http/`)

```go
// Установка куки аутентификации
func SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken string)

// Очистка куки аутентификации
func ClearAuthCookies(w http.ResponseWriter)
```

## ✅ Сильные стороны архитектуры

### 1. **Разделение ответственности**
- ✅ Auth Service отвечает только за аутентификацию
- ✅ GraphQL Server - API Gateway
- ✅ Shared библиотеки для переиспользования

### 2. **Безопасность**
- ✅ JWT с коротким временем жизни (15 мин)
- ✅ Refresh токены с ротацией
- ✅ Поддержка revoke токенов через Redis
- ✅ Валидация email перед входом

### 3. **Масштабируемость**
- ✅ Микросервисная архитектура
- ✅ gRPC для межсервисного взаимодействия
- ✅ Redis для управления сессиями

### 4. **Гибкость**
- ✅ Система ролей с гранулярными правами
- ✅ Поддержка банов и мутов на разных уровнях
- ✅ Типобезопасный контекст

## ⚠️ Потенциальные улучшения

### 1. **Безопасность**
- 🔄 Добавить rate limiting для auth endpoints
- 🔄 Реализовать 2FA (двухфакторную аутентификацию)
- 🔄 Добавить аудит логирование действий пользователей

### 2. **Производительность**
- 🔄 Кэширование прав пользователей
- 🔄 Batch запросы для проверки прав
- 🔄 Оптимизация запросов к базе данных

### 3. **Мониторинг**
- 🔄 Метрики аутентификации (успешные/неуспешные попытки)
- 🔄 Алерты на подозрительную активность
- 🔄 Логирование security events

### 4. **Архитектура**
- 🔄 API Gateway для централизованной аутентификации
- 🔄 Service Mesh для межсервисной безопасности
- 🔄 Централизованное управление конфигурацией

## 🎯 Рекомендации

### 1. **Краткосрочные улучшения**
- Добавить rate limiting
- Улучшить логирование
- Добавить метрики

### 2. **Среднесрочные улучшения**
- Реализовать 2FA
- Добавить аудит
- Оптимизировать производительность

### 3. **Долгосрочные улучшения**
- Service Mesh
- Централизованный API Gateway
- Расширенная система ролей

## 📊 Заключение

Архитектура аутентификации и авторизации в проекте **хорошо спроектирована** и следует современным практикам:

✅ **Микросервисная архитектура** с четким разделением ответственности  
✅ **JWT-based аутентификация** с refresh токенами  
✅ **Гранулярная система ролей** с поддержкой банов и мутов  
✅ **Типобезопасный контекст** для передачи userID  
✅ **Поддержка Redis** для управления сессиями  

Архитектура готова к масштабированию и может быть улучшена добавлением дополнительных слоев безопасности и мониторинга.
