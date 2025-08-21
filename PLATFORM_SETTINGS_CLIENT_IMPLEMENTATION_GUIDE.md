# 🚀 Руководство по реализации настроек платформы на клиенте

## 📋 Обзор

Данное руководство содержит полную информацию о реализованном функционале настроек платформы и инструкции по его реализации на клиентской стороне.

## 🏗️ Архитектура системы

### Основные компоненты:

- **Host (Платформа)** - основная сущность с ID = 1
- **HostRule** - правила платформы
- **HostRole** - роли платформы с правами
- **HostUserMute** - муты пользователей на платформе
- **HostCommunityMute** - муты сообществ на платформе

### Права доступа:

- **Только владелец платформы** может управлять настройками
- Все операции требуют авторизации через JWT токен

## 🔐 Аутентификация

### Получение токена:

```javascript
const loginResponse = await fetch('/query', {
	method: 'POST',
	headers: { 'Content-Type': 'application/json' },
	credentials: 'include',
	body: JSON.stringify({
		query: `mutation { loginUser(input: { email: "admin@example.com", password: "password" }) { user { id name email } } }`,
	}),
})
```

### Проверка владельца платформы:

```javascript
const checkHostOwner = async () => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `query { host { id owner { id name } } }`,
		}),
	})
	const { data } = await response.json()
	return data.host.owner.id === currentUserId
}
```

## 📜 Правила платформы (HostRule)

### Получение всех правил:

```javascript
const getHostRules = async () => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `query { hostRules { id title description createdAt } }`,
		}),
	})
	const { data } = await response.json()
	return data.hostRules
}
```

### Создание правила:

```javascript
const createHostRule = async (title, description) => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `mutation { createHostRule(input: { title: "${title}", description: "${description}" }) { id title description createdAt } }`,
		}),
	})
	const { data } = await response.json()
	return data.createHostRule
}
```

### Обновление правила:

```javascript
const updateHostRule = async (id, title, description) => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `mutation { updateHostRule(input: { id: "${id}", title: "${title}", description: "${description}" }) { id title description updatedAt } }`,
		}),
	})
	const { data } = await response.json()
	return data.updateHostRule
}
```

### Удаление правила:

```javascript
const deleteHostRule = async id => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `mutation { deleteHostRule(id: "${id}") }`,
		}),
	})
	const { data } = await response.json()
	return data.deleteHostRule
}
```

## 👥 Роли платформы (HostRole)

### Получение всех ролей:

```javascript
const getHostRoles = async () => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `query { hostRoles { id title color hostUserBan hostUserMute users { id name email } } }`,
		}),
	})
	const { data } = await response.json()
	return data.hostRoles
}
```

### Создание роли:

```javascript
const createHostRole = async roleData => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `mutation { createHostRole(input: { title: "${roleData.title}", color: "${roleData.color}", hostUserBan: ${roleData.hostUserBan}, hostUserMute: ${roleData.hostUserMute} }) { id title color hostUserBan hostUserMute } }`,
		}),
	})
	const { data } = await response.json()
	return data.createHostRole
}
```

### Добавление пользователя в роль:

```javascript
const addUserToHostRole = async (roleId, userId) => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `mutation { addUserToHostRole(input: { roleID: "${roleId}", userID: "${userId}" }) }`,
		}),
	})
	const { data } = await response.json()
	return data.addUserToHostRole
}
```

### Удаление пользователя из роли:

```javascript
const removeUserFromHostRole = async (roleId, userId) => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `mutation { removeUserFromHostRole(input: { roleID: "${roleId}", userID: "${userId}" }) }`,
		}),
	})
	const { data } = await response.json()
	return data.removeUserFromHostRole
}
```

## 🔇 Муты пользователей (HostUserMute)

### Получение всех мутов:

```javascript
const getHostUserMutes = async () => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `query { hostUserMutes { id userID createdAt user { id name email } } }`,
		}),
	})
	const { data } = await response.json()
	return data.hostUserMutes
}
```

### Мут пользователя:

```javascript
const muteUserOnHost = async userId => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `mutation { muteUserOnHost(input: { userID: "${userId}" }) { id createdAt } }`,
		}),
	})
	const { data } = await response.json()
	return data.muteUserOnHost
}
```

### Размут пользователя:

```javascript
const unmuteUserOnHost = async muteId => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `mutation { unmuteUserOnHost(muteID: "${muteId}") }`,
		}),
	})
	const { data } = await response.json()
	return data.unmuteUserOnHost
}
```

## 🔇 Муты сообществ (HostCommunityMute)

### Получение всех мутов сообществ:

```javascript
const getHostCommunityMutes = async () => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `query { hostCommunityMutes { id communityID createdAt community { id title slug } } }`,
		}),
	})
	const { data } = await response.json()
	return data.hostCommunityMutes
}
```

### Мут сообщества:

```javascript
const muteCommunityOnHost = async communityId => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `mutation { muteCommunityOnHost(input: { communityID: "${communityId}" }) { id communityID createdAt } }`,
		}),
	})
	const { data } = await response.json()
	return data.muteCommunityOnHost
}
```

### Размут сообщества:

```javascript
const unmuteCommunityOnHost = async muteId => {
	const response = await fetch('/query', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		credentials: 'include',
		body: JSON.stringify({
			query: `mutation { unmuteCommunityOnHost(muteID: "${muteId}") }`,
		}),
	})
	const { data } = await response.json()
	return data.unmuteCommunityOnHost
}
```

## 🎨 React Hook для управления правилами

```javascript
import { useState, useEffect } from 'react'

export const useHostRules = () => {
	const [rules, setRules] = useState([])
	const [loading, setLoading] = useState(false)
	const [error, setError] = useState(null)

	const fetchRules = async () => {
		setLoading(true)
		try {
			const rulesData = await getHostRules()
			setRules(rulesData)
		} catch (err) {
			setError(err.message)
		} finally {
			setLoading(false)
		}
	}

	const createRule = async (title, description) => {
		try {
			const newRule = await createHostRule(title, description)
			setRules(prev => [...prev, newRule])
			return newRule
		} catch (err) {
			setError(err.message)
			throw err
		}
	}

	const updateRule = async (id, title, description) => {
		try {
			const updatedRule = await updateHostRule(id, title, description)
			setRules(prev => prev.map(rule => (rule.id === id ? updatedRule : rule)))
			return updatedRule
		} catch (err) {
			setError(err.message)
			throw err
		}
	}

	const deleteRule = async id => {
		try {
			await deleteHostRule(id)
			setRules(prev => prev.filter(rule => rule.id !== id))
		} catch (err) {
			setError(err.message)
			throw err
		}
	}

	useEffect(() => {
		fetchRules()
	}, [])

	return {
		rules,
		loading,
		error,
		createRule,
		updateRule,
		deleteRule,
		refresh: fetchRules,
	}
}
```

## 🎨 React Hook для управления ролями

```javascript
export const useHostRoles = () => {
	const [roles, setRoles] = useState([])
	const [loading, setLoading] = useState(false)
	const [error, setError] = useState(null)

	const fetchRoles = async () => {
		setLoading(true)
		try {
			const rolesData = await getHostRoles()
			setRoles(rolesData)
		} catch (err) {
			setError(err.message)
		} finally {
			setLoading(false)
		}
	}

	const createRole = async roleData => {
		try {
			const newRole = await createHostRole(roleData)
			setRoles(prev => [...prev, newRole])
			return newRole
		} catch (err) {
			setError(err.message)
			throw err
		}
	}

	const addUserToRole = async (roleId, userId) => {
		try {
			await addUserToHostRole(roleId, userId)
			await fetchRoles()
		} catch (err) {
			setError(err.message)
			throw err
		}
	}

	const removeUserFromRole = async (roleId, userId) => {
		try {
			await removeUserFromHostRole(roleId, userId)
			await fetchRoles()
		} catch (err) {
			setError(err.message)
			throw err
		}
	}

	useEffect(() => {
		fetchRoles()
	}, [])

	return {
		roles,
		loading,
		error,
		createRole,
		addUserToRole,
		removeUserFromRole,
		refresh: fetchRoles,
	}
}
```

## 🎨 Компонент управления правилами

```javascript
const HostRulesManager = () => {
	const { rules, loading, createRule, updateRule, deleteRule } = useHostRules()
	const [editingRule, setEditingRule] = useState(null)
	const [formData, setFormData] = useState({ title: '', description: '' })

	const handleSubmit = async e => {
		e.preventDefault()
		try {
			if (editingRule) {
				await updateRule(editingRule.id, formData.title, formData.description)
				setEditingRule(null)
			} else {
				await createRule(formData.title, formData.description)
			}
			setFormData({ title: '', description: '' })
		} catch (error) {
			console.error('Error saving rule:', error)
		}
	}

	if (loading) return <div>Загрузка правил...</div>

	return (
		<div className='host-rules-manager'>
			<h2>Правила платформы</h2>

			<form onSubmit={handleSubmit}>
				<input
					type='text'
					placeholder='Название правила'
					value={formData.title}
					onChange={e =>
						setFormData(prev => ({ ...prev, title: e.target.value }))
					}
					required
				/>
				<textarea
					placeholder='Описание правила'
					value={formData.description}
					onChange={e =>
						setFormData(prev => ({ ...prev, description: e.target.value }))
					}
					required
				/>
				<button type='submit'>
					{editingRule ? 'Обновить' : 'Создать'} правило
				</button>
			</form>

			<div className='rules-list'>
				{rules.map(rule => (
					<div key={rule.id} className='rule-item'>
						<h3>{rule.title}</h3>
						<p>{rule.description}</p>
						<div className='rule-actions'>
							<button onClick={() => setEditingRule(rule)}>
								Редактировать
							</button>
							<button onClick={() => deleteRule(rule.id)}>Удалить</button>
						</div>
					</div>
				))}
			</div>
		</div>
	)
}
```

## 🔧 Обработка ошибок

```javascript
const handleApiError = error => {
	if (error.message.includes('forbidden')) {
		return 'У вас нет прав для выполнения этого действия'
	}
	if (error.message.includes('unauthenticated')) {
		return 'Необходима авторизация'
	}
	return 'Произошла ошибка при выполнении операции'
}
```

## 📋 Чек-лист реализации

- [ ] Настроена аутентификация с cookies
- [ ] Реализована проверка владельца платформы
- [ ] Создан интерфейс управления правилами
- [ ] Создан интерфейс управления ролями
- [ ] Реализовано управление пользователями в ролях
- [ ] Добавлен интерфейс для мутов
- [ ] Настроена обработка ошибок
- [ ] Добавлена валидация форм
- [ ] Реализован адаптивный дизайн

## 🎯 Заключение

Данное руководство содержит все необходимые компоненты для реализации полного функционала настроек платформы на клиентской стороне. Все API endpoints протестированы и готовы к использованию.

**Важно:** Все операции требуют прав владельца платформы и авторизации.
