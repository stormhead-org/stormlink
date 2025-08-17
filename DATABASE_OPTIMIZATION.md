# Оптимизация PostgreSQL и исправление "too many clients already"

## Проблема

Ошибка `pq: sorry, too many clients already` возникла из-за:

1. Отсутствия настроек пула соединений PostgreSQL
2. Избыточных eager-loading запросов в комментариях (`WithAuthor()`, `WithCommunity()`, `WithPost()`)
3. Каскадных запросов к БД в GraphQL резолверах

## Исправления

### 1. Настройка пула соединений PostgreSQL

В `server/cmd/modules/database.go` добавлены настройки пула:

```go
// Переменные окружения для контроля пула соединений
DB_MAX_OPEN_CONNS=15          # Максимальное количество открытых соединений (по умолчанию 15)
DB_MAX_IDLE_CONNS=5           # Максимальное количество idle соединений (по умолчанию 5)
DB_CONN_MAX_LIFETIME_MINUTES=5 # Время жизни соединения в минутах (по умолчанию 5)
```

**Рекомендуемые настройки для production:**

- `DB_MAX_OPEN_CONNS=25` - для средней нагрузки
- `DB_MAX_IDLE_CONNS=10` - для быстрого переиспользования соединений
- `DB_CONN_MAX_LIFETIME_MINUTES=10` - для стабильности

### 2. Оптимизация CommentUsecase

Добавлен новый метод `GetCommentsByPostIDLight()` который:

- Убирает `WithAuthor()`, `WithCommunity()`, `WithPost()`
- Оставляет только `WithMedia()` для URL медиафайлов
- Снижает количество JOIN-запросов к PostgreSQL

### 3. Обновление GraphQL резолвера

`CommentsByPostID` резолвер теперь использует облегченную версию:

```go
return r.CommentUC.GetCommentsByPostIDLight(ctx, pid, hasDeleted)
```

## Использование

### Для клиентских запросов (минимальная нагрузка)

```graphql
query GetCommentsByPostId($id: String!) {
	commentsByPostId(id: $id) {
		id
		content
		createdAt
		authorId
		parentCommentId
		media {
			id
			url
		}
	}
}
```

### Для админки (полная информация)

Используйте `GetCommentsByPostID()` напрямую в usecase для получения всех связанных данных.

## Monitoring

Добавлены логи для отслеживания настроек пула:

```
📊 Настройки пула БД: MaxOpen=15, MaxIdle=5, MaxLifetime=5м
```

## Дополнительные рекомендации

1. **Мониторинг PostgreSQL:**

   ```sql
   SELECT count(*) as current_connections
   FROM pg_stat_activity
   WHERE state = 'active';
   ```

2. **Увеличение max_connections в PostgreSQL** (если нужно):

   ```sql
   ALTER SYSTEM SET max_connections = 200;
   SELECT pg_reload_conf();
   ```

3. **DataLoader для GraphQL** - рассмотрите использование для батч-загрузки связанных данных

4. **Кэширование Redis** - добавьте кэш для часто запрашиваемых комментариев
