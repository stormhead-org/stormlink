# 📚 Документация по работе с правилами сообщества

## 📋 Обзор

Данная документация описывает полный функционал работы с правилами сообщества через GraphQL API. Правила сообщества позволяют администраторам устанавливать и управлять поведением участников в рамках конкретного сообщества.

## 🔐 Авторизация

### Важно: Система авторизации работает через куки

В отличие от стандартных JWT токенов, которые возвращаются в ответе, наша система использует **куки-авторизацию** для безопасности:

1. **Токены НЕ возвращаются** в GraphQL ответах
2. **Куки устанавливаются автоматически** при авторизации
3. **Браузер автоматически отправляет** куки в последующих запросах

### Процесс авторизации:

```javascript
// 1. Авторизация пользователя
const loginResponse = await fetch('/query', {
	method: 'POST',
	headers: {
		'Content-Type': 'application/json',
	},
	credentials: 'include', // Важно! Для работы с куки
	body: JSON.stringify({
		query: `
      mutation {
        loginUser(input: { 
          email: "user@example.com", 
          password: "password" 
        }) {
          user {
            id
            name
            email
          }
        }
      }
    `,
	}),
})

// 2. Куки автоматически устанавливаются браузером
// 3. Последующие запросы автоматически включают куки
```

## 📊 GraphQL Схема

### Типы данных

```graphql
type CommunityRule implements Node {
	id: ID!
	communityID: ID
	title: String!
	description: String
	createdAt: Time!
	updatedAt: Time!
	community: Community
}

input CreateCommunityRuleInput {
	communityID: ID!
	title: String!
	description: String!
}

input UpdateCommunityRuleInput {
	id: ID!
	title: String
	description: String
}
```

## 🚀 Операции с правилами

### 1. Получение списка правил сообщества

```javascript
const getCommunityRules = async communityId => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		credentials: 'include',
		body: JSON.stringify({
			query: `
        query GetCommunityRules($communityID: ID!) {
          communityRules(communityID: $communityID) {
            id
            title
            description
            createdAt
            updatedAt
            community {
              id
              title
            }
          }
        }
      `,
			variables: {
				communityID: communityId,
			},
		}),
	})

	const data = await response.json()
	return data.data.communityRules
}

// Использование
const rules = await getCommunityRules('1')
console.log('Правила сообщества:', rules)
```

### 2. Получение конкретного правила

```javascript
const getCommunityRule = async ruleId => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		credentials: 'include',
		body: JSON.stringify({
			query: `
        query GetCommunityRule($id: ID!) {
          communityRule(id: $id) {
            id
            title
            description
            createdAt
            updatedAt
            community {
              id
              title
            }
          }
        }
      `,
			variables: {
				id: ruleId,
			},
		}),
	})

	const data = await response.json()
	return data.data.communityRule
}

// Использование
const rule = await getCommunityRule('3')
console.log('Правило:', rule)
```

### 3. Создание нового правила

```javascript
const createCommunityRule = async (communityId, title, description) => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		credentials: 'include',
		body: JSON.stringify({
			query: `
        mutation CreateCommunityRule($input: CreateCommunityRuleInput!) {
          createCommunityRule(input: $input) {
            id
            title
            description
            createdAt
            community {
              id
              title
            }
          }
        }
      `,
			variables: {
				input: {
					communityID: communityId,
					title: title,
					description: description,
				},
			},
		}),
	})

	const data = await response.json()

	if (data.errors) {
		throw new Error(data.errors[0].message)
	}

	return data.data.createCommunityRule
}

// Использование
try {
	const newRule = await createCommunityRule(
		'1',
		'Уважение к участникам',
		'Запрещены оскорбления и дискриминация'
	)
	console.log('Создано правило:', newRule)
} catch (error) {
	console.error('Ошибка создания правила:', error.message)
}
```

### 4. Обновление правила

```javascript
const updateCommunityRule = async (ruleId, title, description) => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		credentials: 'include',
		body: JSON.stringify({
			query: `
        mutation UpdateCommunityRule($input: UpdateCommunityRuleInput!) {
          updateCommunityRule(input: $input) {
            id
            title
            description
            createdAt
            updatedAt
          }
        }
      `,
			variables: {
				input: {
					id: ruleId,
					title: title,
					description: description,
				},
			},
		}),
	})

	const data = await response.json()

	if (data.errors) {
		throw new Error(data.errors[0].message)
	}

	return data.data.updateCommunityRule
}

// Использование
try {
	const updatedRule = await updateCommunityRule(
		'3',
		'Обновленное правило',
		'Новое описание правила'
	)
	console.log('Обновлено правило:', updatedRule)
} catch (error) {
	console.error('Ошибка обновления правила:', error.message)
}
```

### 5. Удаление правила

```javascript
const deleteCommunityRule = async ruleId => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		credentials: 'include',
		body: JSON.stringify({
			query: `
        mutation DeleteCommunityRule($id: ID!) {
          deleteCommunityRule(id: $id)
        }
      `,
			variables: {
				id: ruleId,
			},
		}),
	})

	const data = await response.json()

	if (data.errors) {
		throw new Error(data.errors[0].message)
	}

	return data.data.deleteCommunityRule
}

// Использование
try {
	const isDeleted = await deleteCommunityRule('3')
	if (isDeleted) {
		console.log('Правило успешно удалено')
	}
} catch (error) {
	console.error('Ошибка удаления правила:', error.message)
}
```

## 🔧 Утилиты и хелперы

### Класс для работы с правилами

```javascript
class CommunityRulesManager {
	constructor(baseUrl = '/query') {
		this.baseUrl = baseUrl
	}

	async executeQuery(query, variables = {}) {
		const response = await fetch(this.baseUrl, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			credentials: 'include',
			body: JSON.stringify({
				query,
				variables,
			}),
		})

		const data = await response.json()

		if (data.errors) {
			throw new Error(data.errors[0].message)
		}

		return data.data
	}

	// Получить все правила сообщества
	async getRules(communityId) {
		const query = `
      query GetCommunityRules($communityID: ID!) {
        communityRules(communityID: $communityID) {
          id
          title
          description
          createdAt
          updatedAt
        }
      }
    `

		const data = await this.executeQuery(query, { communityID: communityId })
		return data.communityRules
	}

	// Получить конкретное правило
	async getRule(ruleId) {
		const query = `
      query GetCommunityRule($id: ID!) {
        communityRule(id: $id) {
          id
          title
          description
          createdAt
          updatedAt
        }
      }
    `

		const data = await this.executeQuery(query, { id: ruleId })
		return data.communityRule
	}

	// Создать правило
	async createRule(communityId, title, description) {
		const query = `
      mutation CreateCommunityRule($input: CreateCommunityRuleInput!) {
        createCommunityRule(input: $input) {
          id
          title
          description
          createdAt
        }
      }
    `

		const data = await this.executeQuery(query, {
			input: {
				communityID: communityId,
				title,
				description,
			},
		})

		return data.createCommunityRule
	}

	// Обновить правило
	async updateRule(ruleId, title, description) {
		const query = `
      mutation UpdateCommunityRule($input: UpdateCommunityRuleInput!) {
        updateCommunityRule(input: $input) {
          id
          title
          description
          updatedAt
        }
      }
    `

		const data = await this.executeQuery(query, {
			input: {
				id: ruleId,
				title,
				description,
			},
		})

		return data.updateCommunityRule
	}

	// Удалить правило
	async deleteRule(ruleId) {
		const query = `
      mutation DeleteCommunityRule($id: ID!) {
        deleteCommunityRule(id: $id)
      }
    `

		const data = await this.executeQuery(query, { id: ruleId })
		return data.deleteCommunityRule
	}
}

// Использование класса
const rulesManager = new CommunityRulesManager()

// Примеры использования
async function example() {
	try {
		// Получить все правила
		const rules = await rulesManager.getRules('1')
		console.log('Все правила:', rules)

		// Создать новое правило
		const newRule = await rulesManager.createRule(
			'1',
			'Новое правило',
			'Описание нового правила'
		)
		console.log('Создано:', newRule)

		// Обновить правило
		const updatedRule = await rulesManager.updateRule(
			newRule.id,
			'Обновленное правило',
			'Новое описание'
		)
		console.log('Обновлено:', updatedRule)

		// Удалить правило
		const deleted = await rulesManager.deleteRule(newRule.id)
		console.log('Удалено:', deleted)
	} catch (error) {
		console.error('Ошибка:', error.message)
	}
}
```

## ⚠️ Обработка ошибок

### Типичные ошибки и их обработка

```javascript
async function handleCommunityRulesOperation(operation) {
	try {
		const result = await operation()
		return { success: true, data: result }
	} catch (error) {
		// Анализ типа ошибки
		if (error.message.includes('insufficient permissions')) {
			return {
				success: false,
				error: 'Недостаточно прав для выполнения операции',
				code: 'PERMISSION_DENIED',
			}
		}

		if (error.message.includes('not found')) {
			return {
				success: false,
				error: 'Правило не найдено',
				code: 'NOT_FOUND',
			}
		}

		if (error.message.includes('unauthorized')) {
			return {
				success: false,
				error: 'Необходима авторизация',
				code: 'UNAUTHORIZED',
			}
		}

		return {
			success: false,
			error: error.message,
			code: 'UNKNOWN_ERROR',
		}
	}
}

// Использование
const result = await handleCommunityRulesOperation(() =>
	rulesManager.createRule('1', 'Тест', 'Описание')
)

if (result.success) {
	console.log('Операция выполнена:', result.data)
} else {
	console.error('Ошибка:', result.error)
	// Показать пользователю соответствующее сообщение
}
```

## 🔒 Права доступа

### Проверка прав пользователя

Для выполнения операций с правилами сообщества пользователь должен иметь соответствующие права:

- **Создание правил**: Право `communityRolesManagement`
- **Обновление правил**: Право `communityRolesManagement`
- **Удаление правил**: Право `communityRolesManagement`
- **Чтение правил**: Доступно всем авторизованным пользователям

### Проверка роли пользователя

```javascript
const checkUserPermissions = async communityId => {
	const query = `
    query GetMe {
      getMe {
        id
        name
        communitiesRoles {
          id
          title
          communityRolesManagement
          community {
            id
          }
        }
      }
    }
  `

	const response = await fetch('/query', {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		},
		credentials: 'include',
		body: JSON.stringify({ query }),
	})

	const data = await response.json()
	const user = data.data.getMe

	// Проверяем права на конкретное сообщество
	const communityRole = user.communitiesRoles.find(
		role => role.community.id === communityId
	)

	return {
		canManageRules: communityRole?.communityRolesManagement || false,
		roleTitle: communityRole?.title || 'Участник',
	}
}

// Использование
const permissions = await checkUserPermissions('1')
if (permissions.canManageRules) {
	console.log('Пользователь может управлять правилами')
} else {
	console.log('Недостаточно прав для управления правилами')
}
```

## 📱 React Hook для работы с правилами

```javascript
import { useState, useEffect, useCallback } from 'react'

export function useCommunityRules(communityId) {
	const [rules, setRules] = useState([])
	const [loading, setLoading] = useState(false)
	const [error, setError] = useState(null)

	const rulesManager = new CommunityRulesManager()

	// Загрузка правил
	const loadRules = useCallback(async () => {
		setLoading(true)
		setError(null)

		try {
			const rulesData = await rulesManager.getRules(communityId)
			setRules(rulesData)
		} catch (err) {
			setError(err.message)
		} finally {
			setLoading(false)
		}
	}, [communityId])

	// Создание правила
	const createRule = useCallback(
		async (title, description) => {
			try {
				const newRule = await rulesManager.createRule(
					communityId,
					title,
					description
				)
				setRules(prev => [...prev, newRule])
				return { success: true, data: newRule }
			} catch (err) {
				return { success: false, error: err.message }
			}
		},
		[communityId]
	)

	// Обновление правила
	const updateRule = useCallback(async (ruleId, title, description) => {
		try {
			const updatedRule = await rulesManager.updateRule(
				ruleId,
				title,
				description
			)
			setRules(prev =>
				prev.map(rule => (rule.id === ruleId ? updatedRule : rule))
			)
			return { success: true, data: updatedRule }
		} catch (err) {
			return { success: false, error: err.message }
		}
	}, [])

	// Удаление правила
	const deleteRule = useCallback(async ruleId => {
		try {
			await rulesManager.deleteRule(ruleId)
			setRules(prev => prev.filter(rule => rule.id !== ruleId))
			return { success: true }
		} catch (err) {
			return { success: false, error: err.message }
		}
	}, [])

	// Загружаем правила при монтировании
	useEffect(() => {
		if (communityId) {
			loadRules()
		}
	}, [communityId, loadRules])

	return {
		rules,
		loading,
		error,
		createRule,
		updateRule,
		deleteRule,
		reloadRules: loadRules,
	}
}

// Использование в компоненте
function CommunityRulesComponent({ communityId }) {
	const { rules, loading, error, createRule, updateRule, deleteRule } =
		useCommunityRules(communityId)

	const handleCreateRule = async () => {
		const result = await createRule('Новое правило', 'Описание')
		if (result.success) {
			alert('Правило создано!')
		} else {
			alert('Ошибка: ' + result.error)
		}
	}

	if (loading) return <div>Загрузка правил...</div>
	if (error) return <div>Ошибка: {error}</div>

	return (
		<div>
			<h2>Правила сообщества</h2>
			<button onClick={handleCreateRule}>Создать правило</button>

			{rules.map(rule => (
				<div key={rule.id}>
					<h3>{rule.title}</h3>
					<p>{rule.description}</p>
					<small>
						Создано: {new Date(rule.createdAt).toLocaleDateString()}
					</small>
				</div>
			))}
		</div>
	)
}
```

## 🧪 Тестирование

### Примеры тестовых запросов

```bash
# 1. Авторизация
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -c cookies.txt \
  -d '{
    "query": "mutation { loginUser(input: { email: \"gamenimsi@gmail.com\", password: \"qqwdqqwd\" }) { user { id name } } }"
  }'

# 2. Получение правил
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "query": "query { communityRules(communityID: \"1\") { id title description } }"
  }'

# 3. Создание правила
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -b cookies.txt \
  -d '{
    "query": "mutation { createCommunityRule(input: { communityID: \"1\", title: \"Тест\", description: \"Описание\" }) { id title } }"
  }'
```

## 📝 Чек-лист для реализации

- [ ] Настроить авторизацию с куки (`credentials: 'include'`)
- [ ] Реализовать обработку ошибок авторизации
- [ ] Добавить проверку прав пользователя
- [ ] Создать UI для отображения правил
- [ ] Добавить формы создания/редактирования правил
- [ ] Реализовать подтверждение удаления
- [ ] Добавить валидацию входных данных
- [ ] Настроить обновление списка после операций
- [ ] Добавить индикаторы загрузки
- [ ] Реализовать пагинацию (если нужно)

## 🔗 Полезные ссылки

- [GraphQL Playground](http://localhost:8080/) - для тестирования запросов
- [Документация по авторизации](./AUTH_ARCHITECTURE_ANALYSIS.md)
- [Руководство по безопасности](./FINAL_SECURITY_REPORT.md)
