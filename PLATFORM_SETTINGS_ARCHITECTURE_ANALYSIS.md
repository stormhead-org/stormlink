# 🏗️ Анализ архитектуры настроек платформы

## 📋 Обзор

Данный документ анализирует существующую архитектуру и предлагает план реализации функционала настроек платформы, аналогичного функционалу сообществ.

## 🔍 Анализ существующей архитектуры

### 1. Модель Host (Платформа)

```go
// server/ent/schema/host.go
type Host struct {
    ID            int
    Title         string
    Slogan        string
    Contacts      string
    Description   string
    LogoID        int
    BannerID      int
    AuthBannerID  int
    OwnerID       int
    FirstSettings bool
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

**Особенности:**

- ✅ **Фиксированный ID = 1** - всегда используется хост с ID 1
- ✅ **Единственный владелец** - owner_id указывает на владельца платформы
- ✅ **Связи с медиа** - логотип, баннер, баннер авторизации
- ✅ **Связь с правилами** - edge.To("rules", HostRule.Type)

### 2. Существующие схемы платформы

#### HostRule (Правила платформы)

```go
type HostRule struct {
    ID              int
    RuleID          int      // Связь с хостом
    NameRule        string   // Название правила
    DescriptionRule string   // Описание правила
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

#### HostUserBan (Бан пользователей на платформе)

```go
type HostUserBan struct {
    ID        int
    CreatedAt time.Time
    UpdatedAt time.Time
    // Связь с пользователем через edge
}
```

#### HostCommunityBan (Бан сообществ на платформе)

```go
type HostCommunityBan struct {
    ID          int
    CommunityID int
    CreatedAt   time.Time
    UpdatedAt   time.Time
    // Связь с сообществом через edge
}
```

#### HostUserMute (Мут пользователей на платформе)

```go
type HostUserMute struct {
    ID        int
    CreatedAt time.Time
    UpdatedAt time.Time
    // Связь с пользователем через edge
}
```

#### HostCommunityMute (Мут сообществ на платформе)

```go
type HostCommunityMute struct {
    ID          int
    CommunityID int
    CreatedAt   time.Time
    UpdatedAt   time.Time
    // Связь с сообществом через edge
}
```

### 3. Существующие GraphQL резолверы

#### Query резолверы

```go
// Получение хоста (всегда ID = 1)
func (r *queryResolver) Host(ctx context.Context) (*ent.Host, error) {
    return r.Client.Host.Get(ctx, 1)
}

// Роли хоста
func (r *queryResolver) HostRole(ctx context.Context, id string) (*ent.HostRole, error)
func (r *queryResolver) HostRoles(ctx context.Context) ([]*ent.HostRole, error)

// Баны хоста
func (r *queryResolver) HostUserBan(ctx context.Context, id string) (*ent.HostUserBan, error)
func (r *queryResolver) HostUsersBan(ctx context.Context) ([]*ent.HostUserBan, error)
func (r *queryResolver) HostCommunityBans(ctx context.Context) ([]*models.HostCommunityBan, error)
func (r *queryResolver) HostCommunityBan(ctx context.Context, id string) (*models.HostCommunityBan, error)
```

#### Mutation резолверы

```go
// Обновление настроек хоста
func (r *mutationResolver) Host(ctx context.Context, input models.UpdateHostInput) (*ent.Host, error)

// Управление ролями хоста
func (r *mutationResolver) CreateHostRole(ctx context.Context, input models.CreateHostRoleInput) (*ent.HostRole, error)
func (r *mutationResolver) UpdateHostRole(ctx context.Context, input models.UpdateHostRoleInput) (*ent.HostRole, error)
func (r *mutationResolver) DeleteHostRole(ctx context.Context, id string) (bool, error)

// Баны хоста
func (r *mutationResolver) BanUserFromHost(ctx context.Context, input models.BanUserInput) (*ent.HostUserBan, error)
func (r *mutationResolver) UnbanUserFromHost(ctx context.Context, banID string) (bool, error)
func (r *mutationResolver) BanCommunityFromHost(ctx context.Context, input models.BanCommunityInput) (*models.HostCommunityBan, error)
func (r *mutationResolver) UnbanCommunityFromHost(ctx context.Context, banID string) (bool, error)
```

## 🎯 План реализации недостающего функционала

### 1. Правила платформы (HostRule)

#### Проблемы текущей реализации:

- ❌ **Нет GraphQL схемы** для правил хоста
- ❌ **Нет резолверов** для CRUD операций
- ❌ **Нет usecase** для бизнес-логики
- ❌ **Неправильная схема** - rule_id вместо host_id

#### Необходимые изменения:

1. **Исправить схему HostRule:**

```go
type HostRule struct {
    ID          int
    HostID      int      // Связь с хостом (всегда 1)
    Title       string   // Название правила
    Description string   // Описание правила
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

2. **Добавить GraphQL схему:**

```graphql
# Queries
hostRules: [HostRule!]!
hostRule(id: ID!): HostRule

# Mutations
createHostRule(input: CreateHostRuleInput!): HostRule!
updateHostRule(input: UpdateHostRuleInput!): HostRule!
deleteHostRule(id: ID!): Boolean!

# Input types
input CreateHostRuleInput {
    title: String!
    description: String!
}

input UpdateHostRuleInput {
    id: ID!
    title: String
    description: String
}
```

3. **Создать usecase:**

```go
type HostRuleUsecase interface {
    CreateHostRule(ctx context.Context, input *models.CreateHostRuleInput) (*ent.HostRule, error)
    UpdateHostRule(ctx context.Context, input *models.UpdateHostRuleInput) (*ent.HostRule, error)
    DeleteHostRule(ctx context.Context, id string) (bool, error)
    GetHostRule(ctx context.Context, id string) (*ent.HostRule, error)
    GetHostRules(ctx context.Context) ([]*ent.HostRule, error)
}
```

### 2. Муты платформы (HostUserMute, HostCommunityMute)

#### Проблемы:

- ❌ **Нет GraphQL схемы** для мутов
- ❌ **Нет резолверов** для управления мутами
- ❌ **Нет usecase** для бизнес-логики

#### Необходимые изменения:

1. **Добавить GraphQL схему:**

```graphql
# Queries
hostUserMutes: [HostUserMute!]!
hostUserMute(id: ID!): HostUserMute
hostCommunityMutes: [HostCommunityMute!]!
hostCommunityMute(id: ID!): HostCommunityMute

# Mutations
muteUserOnHost(input: MuteUserInput!): HostUserMute!
unmuteUserOnHost(muteID: ID!): Boolean!
muteCommunityOnHost(input: MuteCommunityInput!): HostCommunityMute!
unmuteCommunityOnHost(muteID: ID!): Boolean!

# Input types
input MuteUserInput {
    userID: ID!
}

input MuteCommunityInput {
    communityID: ID!
}
```

2. **Создать usecase:**

```go
type HostMuteUsecase interface {
    MuteUser(ctx context.Context, userID string) (*ent.HostUserMute, error)
    UnmuteUser(ctx context.Context, muteID string) (bool, error)
    GetUserMutes(ctx context.Context) ([]*ent.HostUserMute, error)
    MuteCommunity(ctx context.Context, communityID string) (*ent.HostCommunityMute, error)
    UnmuteCommunity(ctx context.Context, muteID string) (bool, error)
    GetCommunityMutes(ctx context.Context) ([]*ent.HostCommunityMute, error)
}
```

### 3. Права доступа

#### Текущая система прав:

- ✅ **HostRole** - роли платформы с правами
- ✅ **Права ролей:**
  - `hostUserBan` - бан пользователей
  - `hostUserMute` - мут пользователей
  - `hostCommunityDeletePost` - удаление постов
  - `hostCommunityRemovePostFromPublication` - снятие с публикации
  - `hostCommunityDeleteComments` - удаление комментариев

#### Необходимые дополнения:

```go
type HostRole struct {
    // ... существующие поля
    HostRulesManagement     bool // Управление правилами платформы
    HostCommunityBan        bool // Бан сообществ
    HostCommunityMute       bool // Мут сообществ
}
```

### 4. Система разрешений

#### Логика проверки прав:

```go
func (uc *hostRuleUsecase) canManageHostRules(ctx context.Context, userID int) (bool, error) {
    // 1. Проверяем, является ли пользователь владельцем платформы
    host, err := uc.client.Host.Get(ctx, 1)
    if err != nil {
        return false, err
    }

    if host.OwnerID == userID {
        return true, nil
    }

    // 2. Проверяем роли пользователя на платформе
    roles, err := uc.client.HostRole.
        Query().
        Where(hostrole.HasUsersWith(user.IDEQ(userID))).
        All(ctx)

    for _, role := range roles {
        if role.HostRulesManagement {
            return true, nil
        }
    }

    return false, nil
}
```

## 📊 Сравнение архитектур

| Функционал            | Сообщества        | Платформа         |
| --------------------- | ----------------- | ----------------- |
| **Основная сущность** | Community         | Host (ID=1)       |
| **Владелец**          | Community.OwnerID | Host.OwnerID      |
| **Правила**           | CommunityRule     | HostRule          |
| **Роли**              | Role              | HostRole          |
| **Бан пользователей** | CommunityUserBan  | HostUserBan       |
| **Мут пользователей** | CommunityUserMute | HostUserMute      |
| **Бан сообществ**     | -                 | HostCommunityBan  |
| **Мут сообществ**     | -                 | HostCommunityMute |

## 🚀 План реализации

### Этап 1: Исправление схемы HostRule

1. Обновить схему `server/ent/schema/host_rule.go`
2. Сгенерировать ent код
3. Создать миграцию

### Этап 2: GraphQL схема

1. Добавить типы в `server/graphql/handlers.graphql`
2. Сгенерировать GraphQL код

### Этап 3: Usecase слой

1. Создать `server/usecase/hostrule/`
2. Создать `server/usecase/hostmute/`
3. Реализовать бизнес-логику

### Этап 4: Резолверы

1. Добавить query резолверы
2. Добавить mutation резолверы
3. Интегрировать usecase

### Этап 5: Тестирование

1. Создать тесты для usecase
2. Создать интеграционные тесты
3. Создать документацию

## 🔒 Безопасность

### Принципы безопасности:

1. **Фиксированный ID хоста** - всегда используется ID=1
2. **Проверка владельца** - только владелец платформы имеет полные права
3. **Ролевая система** - права через роли платформы
4. **Авторизация** - все операции требуют авторизации
5. **Валидация** - проверка входных данных

### Права доступа:

- **Владелец платформы** - полные права на все операции
- **Роли с правами** - ограниченные права согласно роли
- **Обычные пользователи** - только чтение правил

## 📝 Заключение

Архитектура настроек платформы должна быть аналогична архитектуре сообществ, но с учетом специфики платформы (фиксированный ID=1, единственный владелец). Основные компоненты уже существуют, необходимо добавить недостающий функционал и исправить существующие проблемы.
