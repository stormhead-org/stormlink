# Руководство по реализации функционала правил сообществ

## Обзор

Функционал правил сообществ позволяет владельцам создавать, редактировать и удалять правила для своих сообществ.

## API Endpoints

### Queries

```graphql
# Получить одно правило
query GetCommunityRule($id: ID!) {
	communityRule(id: $id) {
		id
		title
		description
		createdAt
		updatedAt
	}
}

# Получить все правила сообщества
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

### Mutations

```graphql
# Создать правило
mutation CreateCommunityRule($input: CreateCommunityRuleInput!) {
	createCommunityRule(input: $input) {
		id
		title
		description
		createdAt
		updatedAt
	}
}

# Обновить правило
mutation UpdateCommunityRule($input: UpdateCommunityRuleInput!) {
	updateCommunityRule(input: $input) {
		id
		title
		description
		createdAt
		updatedAt
	}
}

# Удалить правило
mutation DeleteCommunityRule($id: ID!) {
	deleteCommunityRule(id: $id)
}
```

## Входные типы

```typescript
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

## Права доступа

- **Создание/редактирование/удаление**: Только владелец сообщества
- **Просмотр**: Все участники сообщества

## Реализация на клиенте

### 1. React Hook

```typescript
export const useCommunityRules = (communityID: string) => {
	const { data, loading, error } = useQuery({
		queryKey: ['communityRules', communityID],
		queryFn: () => getCommunityRules(communityID),
	})

	const createRule = useMutation({
		mutationFn: createCommunityRule,
		onSuccess: () =>
			queryClient.invalidateQueries(['communityRules', communityID]),
	})

	const updateRule = useMutation({
		mutationFn: updateCommunityRule,
		onSuccess: () =>
			queryClient.invalidateQueries(['communityRules', communityID]),
	})

	const deleteRule = useMutation({
		mutationFn: deleteCommunityRule,
		onSuccess: () =>
			queryClient.invalidateQueries(['communityRules', communityID]),
	})

	return {
		rules: data?.communityRules || [],
		loading,
		error,
		createRule: createRule.mutate,
		updateRule: updateRule.mutate,
		deleteRule: deleteRule.mutate,
	}
}
```

### 2. Компонент списка правил

```typescript
export const CommunityRulesList = ({ communityID }) => {
	const { rules, loading, deleteRule } = useCommunityRules(communityID)
	const { permissions } = useCommunityPermissions(communityID)

	const canManageRules = permissions?.communityOwner

	if (loading) return <div>Загрузка...</div>

	return (
		<div>
			<h3>Правила сообщества</h3>
			{rules.map(rule => (
				<div key={rule.id}>
					<h4>{rule.title}</h4>
					<p>{rule.description}</p>
					{canManageRules && (
						<button onClick={() => deleteRule(rule.id)}>Удалить</button>
					)}
				</div>
			))}
			{canManageRules && (
				<button onClick={() => setShowCreateModal(true)}>
					Добавить правило
				</button>
			)}
		</div>
	)
}
```

### 3. Модальное окно создания/редактирования

```typescript
export const RuleFormModal = ({ communityID, rule, isOpen, onClose }) => {
	const [title, setTitle] = useState(rule?.title || '')
	const [description, setDescription] = useState(rule?.description || '')
	const { createRule, updateRule } = useCommunityRules(communityID)

	const handleSubmit = async e => {
		e.preventDefault()

		if (rule) {
			await updateRule({ id: rule.id, title, description })
		} else {
			await createRule({ communityID, title, description })
		}

		onClose()
	}

	return (
		<Modal isOpen={isOpen} onClose={onClose}>
			<form onSubmit={handleSubmit}>
				<input
					value={title}
					onChange={e => setTitle(e.target.value)}
					placeholder='Название правила'
					required
				/>
				<textarea
					value={description}
					onChange={e => setDescription(e.target.value)}
					placeholder='Описание правила'
				/>
				<button type='submit'>{rule ? 'Сохранить' : 'Создать'}</button>
			</form>
		</Modal>
	)
}
```

## Интеграция в страницу сообщества

```typescript
export const CommunityPage = () => {
	const [showCreateModal, setShowCreateModal] = useState(false)
	const [editingRule, setEditingRule] = useState(null)

	return (
		<div>
			<CommunityRulesList
				communityID={community.id}
				onEditRule={setEditingRule}
				onCreateRule={() => setShowCreateModal(true)}
			/>

			<RuleFormModal
				communityID={community.id}
				rule={editingRule}
				isOpen={showCreateModal || !!editingRule}
				onClose={() => {
					setShowCreateModal(false)
					setEditingRule(null)
				}}
			/>
		</div>
	)
}
```

## Обработка ошибок

```typescript
const handleRuleError = error => {
	if (error.message.includes('unauthorized')) {
		return 'Необходима авторизация'
	}
	if (error.message.includes('insufficient permissions')) {
		return 'Недостаточно прав'
	}
	return 'Произошла ошибка'
}
```

## Готовые функции для API

```typescript
// api/communityRules.ts
export const getCommunityRules = async (communityID: string) => {
	const response = await client.query({
		query: GET_COMMUNITY_RULES,
		variables: { communityID },
	})
	return response.data.communityRules
}

export const createCommunityRule = async (input: CreateCommunityRuleInput) => {
	const response = await client.mutate({
		mutation: CREATE_COMMUNITY_RULE,
		variables: { input },
	})
	return response.data.createCommunityRule
}

export const updateCommunityRule = async (input: UpdateCommunityRuleInput) => {
	const response = await client.mutate({
		mutation: UPDATE_COMMUNITY_RULE,
		variables: { input },
	})
	return response.data.updateCommunityRule
}

export const deleteCommunityRule = async (id: string) => {
	const response = await client.mutate({
		mutation: DELETE_COMMUNITY_RULE,
		variables: { id },
	})
	return response.data.deleteCommunityRule
}
```

Функционал готов к использованию! Все необходимые API endpoints реализованы на бэкенде.
