# Руководство по реализации функционала правил сообществ

## Обзор

Функционал правил сообществ позволяет владельцам сообществ создавать, редактировать и удалять правила для своих сообществ. Правила отображаются участникам сообщества и помогают поддерживать порядок.

## API Endpoints

### Queries (Запросы)

#### 1. Получение одного правила

```graphql
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
			slug
		}
	}
}
```

#### 2. Получение всех правил сообщества

```graphql
query GetCommunityRules($communityID: ID!) {
	communityRules(communityID: $communityID) {
		id
		title
		description
		createdAt
		updatedAt
	}
}
```

### Mutations (Мутации)

#### 1. Создание правила

```graphql
mutation CreateCommunityRule($input: CreateCommunityRuleInput!) {
	createCommunityRule(input: $input) {
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
```

**Входные данные:**

```typescript
interface CreateCommunityRuleInput {
	communityID: string
	title: string
	description: string
}
```

#### 2. Обновление правила

```graphql
mutation UpdateCommunityRule($input: UpdateCommunityRuleInput!) {
	updateCommunityRule(input: $input) {
		id
		title
		description
		createdAt
		updatedAt
	}
}
```

**Входные данные:**

```typescript
interface UpdateCommunityRuleInput {
	id: string
	title?: string
	description?: string
}
```

#### 3. Удаление правила

```graphql
mutation DeleteCommunityRule($id: ID!) {
	deleteCommunityRule(id: $id)
}
```

## Права доступа

- **Создание, редактирование, удаление**: Только владелец сообщества
- **Просмотр**: Все участники сообщества

## Реализация на клиенте

### 1. Типы TypeScript

```typescript
interface CommunityRule {
	id: string
	title: string
	description: string
	createdAt: string
	updatedAt: string
	community?: {
		id: string
		title: string
		slug: string
	}
}

interface CreateCommunityRuleInput {
	communityID: string
	title: string
	description: string
}

interface UpdateCommunityRuleInput {
	id: string
	title?: string
	description?: string
}
```

### 2. React Hook для работы с правилами

```typescript
// hooks/useCommunityRules.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { gql } from '@apollo/client'

const GET_COMMUNITY_RULES = gql`
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

const CREATE_COMMUNITY_RULE = gql`
	mutation CreateCommunityRule($input: CreateCommunityRuleInput!) {
		createCommunityRule(input: $input) {
			id
			title
			description
			createdAt
			updatedAt
		}
	}
`

const UPDATE_COMMUNITY_RULE = gql`
	mutation UpdateCommunityRule($input: UpdateCommunityRuleInput!) {
		updateCommunityRule(input: $input) {
			id
			title
			description
			createdAt
			updatedAt
		}
	}
`

const DELETE_COMMUNITY_RULE = gql`
	mutation DeleteCommunityRule($id: ID!) {
		deleteCommunityRule(id: $id)
	}
`

export const useCommunityRules = (communityID: string) => {
	const queryClient = useQueryClient()

	const { data, loading, error } = useQuery({
		queryKey: ['communityRules', communityID],
		queryFn: () =>
			client.query({
				query: GET_COMMUNITY_RULES,
				variables: { communityID },
			}),
		enabled: !!communityID,
	})

	const createRule = useMutation({
		mutationFn: (input: CreateCommunityRuleInput) =>
			client.mutate({
				mutation: CREATE_COMMUNITY_RULE,
				variables: { input },
			}),
		onSuccess: () => {
			queryClient.invalidateQueries(['communityRules', communityID])
		},
	})

	const updateRule = useMutation({
		mutationFn: (input: UpdateCommunityRuleInput) =>
			client.mutate({
				mutation: UPDATE_COMMUNITY_RULE,
				variables: { input },
			}),
		onSuccess: () => {
			queryClient.invalidateQueries(['communityRules', communityID])
		},
	})

	const deleteRule = useMutation({
		mutationFn: (id: string) =>
			client.mutate({
				mutation: DELETE_COMMUNITY_RULE,
				variables: { id },
			}),
		onSuccess: () => {
			queryClient.invalidateQueries(['communityRules', communityID])
		},
	})

	return {
		rules: data?.communityRules || [],
		loading,
		error,
		createRule: createRule.mutate,
		updateRule: updateRule.mutate,
		deleteRule: deleteRule.mutate,
		isCreating: createRule.isPending,
		isUpdating: updateRule.isPending,
		isDeleting: deleteRule.isPending,
	}
}
```

### 3. Компонент списка правил

```typescript
// components/CommunityRulesList.tsx
import React from 'react'
import { useCommunityRules } from '../hooks/useCommunityRules'
import { useAuth } from '../hooks/useAuth'
import { useCommunityPermissions } from '../hooks/useCommunityPermissions'

interface CommunityRulesListProps {
	communityID: string
}

export const CommunityRulesList: React.FC<CommunityRulesListProps> = ({
	communityID,
}) => {
	const { rules, loading, error, deleteRule, isDeleting } =
		useCommunityRules(communityID)
	const { user } = useAuth()
	const { permissions } = useCommunityPermissions(communityID)

	const canManageRules = permissions?.communityOwner || false

	if (loading) return <div>Загрузка правил...</div>
	if (error) return <div>Ошибка загрузки правил: {error.message}</div>

	return (
		<div className='community-rules'>
			<h3>Правила сообщества</h3>

			{rules.length === 0 ? (
				<p>Правила не установлены</p>
			) : (
				<div className='rules-list'>
					{rules.map(rule => (
						<div key={rule.id} className='rule-item'>
							<h4>{rule.title}</h4>
							<p>{rule.description}</p>
							<small>
								Создано: {new Date(rule.createdAt).toLocaleDateString()}
							</small>

							{canManageRules && (
								<div className='rule-actions'>
									<button
										onClick={() => handleEditRule(rule)}
										className='btn btn-sm btn-outline'
									>
										Редактировать
									</button>
									<button
										onClick={() => handleDeleteRule(rule.id)}
										disabled={isDeleting}
										className='btn btn-sm btn-danger'
									>
										{isDeleting ? 'Удаление...' : 'Удалить'}
									</button>
								</div>
							)}
						</div>
					))}
				</div>
			)}

			{canManageRules && (
				<button
					onClick={() => setShowCreateModal(true)}
					className='btn btn-primary'
				>
					Добавить правило
				</button>
			)}
		</div>
	)
}
```

### 4. Модальное окно создания/редактирования правила

```typescript
// components/RuleFormModal.tsx
import React, { useState, useEffect } from 'react'
import { useCommunityRules } from '../hooks/useCommunityRules'

interface RuleFormModalProps {
	communityID: string
	rule?: CommunityRule
	isOpen: boolean
	onClose: () => void
}

export const RuleFormModal: React.FC<RuleFormModalProps> = ({
	communityID,
	rule,
	isOpen,
	onClose,
}) => {
	const [title, setTitle] = useState('')
	const [description, setDescription] = useState('')
	const { createRule, updateRule, isCreating, isUpdating } =
		useCommunityRules(communityID)

	const isEditing = !!rule
	const isLoading = isCreating || isUpdating

	useEffect(() => {
		if (rule) {
			setTitle(rule.title)
			setDescription(rule.description)
		} else {
			setTitle('')
			setDescription('')
		}
	}, [rule])

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault()

		if (isEditing && rule) {
			await updateRule({
				id: rule.id,
				title,
				description,
			})
		} else {
			await createRule({
				communityID,
				title,
				description,
			})
		}

		onClose()
	}

	if (!isOpen) return null

	return (
		<div className='modal-overlay'>
			<div className='modal'>
				<h3>{isEditing ? 'Редактировать правило' : 'Создать правило'}</h3>

				<form onSubmit={handleSubmit}>
					<div className='form-group'>
						<label htmlFor='title'>Название правила *</label>
						<input
							id='title'
							type='text'
							value={title}
							onChange={e => setTitle(e.target.value)}
							required
							disabled={isLoading}
						/>
					</div>

					<div className='form-group'>
						<label htmlFor='description'>Описание</label>
						<textarea
							id='description'
							value={description}
							onChange={e => setDescription(e.target.value)}
							rows={4}
							disabled={isLoading}
						/>
					</div>

					<div className='modal-actions'>
						<button
							type='button'
							onClick={onClose}
							disabled={isLoading}
							className='btn btn-secondary'
						>
							Отмена
						</button>
						<button
							type='submit'
							disabled={isLoading || !title.trim()}
							className='btn btn-primary'
						>
							{isLoading
								? isEditing
									? 'Сохранение...'
									: 'Создание...'
								: isEditing
								? 'Сохранить'
								: 'Создать'}
						</button>
					</div>
				</form>
			</div>
		</div>
	)
}
```

### 5. Интеграция в страницу сообщества

```typescript
// pages/CommunityPage.tsx
import React, { useState } from 'react'
import { CommunityRulesList } from '../components/CommunityRulesList'
import { RuleFormModal } from '../components/RuleFormModal'

export const CommunityPage: React.FC = () => {
	const [showCreateModal, setShowCreateModal] = useState(false)
	const [editingRule, setEditingRule] = useState<CommunityRule | null>(null)

	const handleEditRule = (rule: CommunityRule) => {
		setEditingRule(rule)
	}

	const handleCloseModal = () => {
		setShowCreateModal(false)
		setEditingRule(null)
	}

	return (
		<div className='community-page'>
			{/* Другие компоненты сообщества */}

			<CommunityRulesList
				communityID={community.id}
				onEditRule={handleEditRule}
				onCreateRule={() => setShowCreateModal(true)}
			/>

			<RuleFormModal
				communityID={community.id}
				rule={editingRule}
				isOpen={showCreateModal || !!editingRule}
				onClose={handleCloseModal}
			/>
		</div>
	)
}
```

## Стили CSS

```css
.community-rules {
	margin: 2rem 0;
}

.rules-list {
	display: flex;
	flex-direction: column;
	gap: 1rem;
}

.rule-item {
	border: 1px solid #e1e5e9;
	border-radius: 8px;
	padding: 1rem;
	background: #fff;
}

.rule-item h4 {
	margin: 0 0 0.5rem 0;
	color: #2c3e50;
}

.rule-item p {
	margin: 0 0 1rem 0;
	color: #7f8c8d;
	line-height: 1.5;
}

.rule-item small {
	color: #95a5a6;
	font-size: 0.875rem;
}

.rule-actions {
	margin-top: 1rem;
	display: flex;
	gap: 0.5rem;
}

.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	background: rgba(0, 0, 0, 0.5);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 1000;
}

.modal {
	background: #fff;
	border-radius: 8px;
	padding: 2rem;
	max-width: 500px;
	width: 90%;
	max-height: 90vh;
	overflow-y: auto;
}

.form-group {
	margin-bottom: 1rem;
}

.form-group label {
	display: block;
	margin-bottom: 0.5rem;
	font-weight: 500;
}

.form-group input,
.form-group textarea {
	width: 100%;
	padding: 0.75rem;
	border: 1px solid #ddd;
	border-radius: 4px;
	font-size: 1rem;
}

.modal-actions {
	display: flex;
	gap: 1rem;
	justify-content: flex-end;
	margin-top: 2rem;
}
```

## Обработка ошибок

```typescript
// utils/errorHandling.ts
export const handleRuleError = (error: any) => {
	if (error.message.includes('unauthorized')) {
		return 'Необходима авторизация'
	}

	if (error.message.includes('insufficient permissions')) {
		return 'Недостаточно прав для выполнения действия'
	}

	if (error.message.includes('rule not found')) {
		return 'Правило не найдено'
	}

	return 'Произошла ошибка. Попробуйте еще раз.'
}
```

## Тестирование

### Unit тесты для хука

```typescript
// hooks/__tests__/useCommunityRules.test.ts
import { renderHook, waitFor } from '@testing-library/react'
import { useCommunityRules } from '../useCommunityRules'

describe('useCommunityRules', () => {
	it('should fetch community rules', async () => {
		const { result } = renderHook(() => useCommunityRules('1'))

		await waitFor(() => {
			expect(result.current.loading).toBe(false)
		})

		expect(result.current.rules).toHaveLength(2)
	})

	it('should create a new rule', async () => {
		const { result } = renderHook(() => useCommunityRules('1'))

		await result.current.createRule({
			communityID: '1',
			title: 'Новое правило',
			description: 'Описание правила',
		})

		expect(result.current.rules).toHaveLength(3)
	})
})
```

## Заключение

Этот функционал предоставляет полный набор возможностей для управления правилами сообществ:

1. **Просмотр правил** - доступен всем участникам
2. **Создание правил** - только владельцу сообщества
3. **Редактирование правил** - только владельцу сообщества
4. **Удаление правил** - только владельцу сообщества

Реализация включает в себя:

- TypeScript типы для типобезопасности
- React hooks для управления состоянием
- Компоненты для отображения и редактирования
- Обработку ошибок и состояний загрузки
- Стили для красивого отображения
- Тесты для проверки функциональности

Функционал готов к использованию и может быть легко интегрирован в существующее приложение.
