# 📖 Примеры использования правил сообщества

## 🎯 Vue.js 3 Composition API

### Композабл для работы с правилами

```javascript
// composables/useCommunityRules.js
import { ref, computed } from 'vue'

export function useCommunityRules(communityId) {
	const rules = ref([])
	const loading = ref(false)
	const error = ref(null)

	const rulesManager = new CommunityRulesManager()

	const loadRules = async () => {
		loading.value = true
		error.value = null

		try {
			const rulesData = await rulesManager.getRules(communityId)
			rules.value = rulesData
		} catch (err) {
			error.value = err.message
		} finally {
			loading.value = false
		}
	}

	const createRule = async (title, description) => {
		try {
			const newRule = await rulesManager.createRule(
				communityId,
				title,
				description
			)
			rules.value.push(newRule)
			return { success: true, data: newRule }
		} catch (err) {
			return { success: false, error: err.message }
		}
	}

	const updateRule = async (ruleId, title, description) => {
		try {
			const updatedRule = await rulesManager.updateRule(
				ruleId,
				title,
				description
			)
			const index = rules.value.findIndex(rule => rule.id === ruleId)
			if (index !== -1) {
				rules.value[index] = updatedRule
			}
			return { success: true, data: updatedRule }
		} catch (err) {
			return { success: false, error: err.message }
		}
	}

	const deleteRule = async ruleId => {
		try {
			await rulesManager.deleteRule(ruleId)
			rules.value = rules.value.filter(rule => rule.id !== ruleId)
			return { success: true }
		} catch (err) {
			return { success: false, error: err.message }
		}
	}

	return {
		rules: computed(() => rules.value),
		loading: computed(() => loading.value),
		error: computed(() => error.value),
		loadRules,
		createRule,
		updateRule,
		deleteRule,
	}
}
```

### Компонент Vue

```vue
<!-- components/CommunityRules.vue -->
<template>
	<div class="community-rules">
		<h2>Правила сообщества</h2>

		<!-- Форма создания -->
		<div class="create-form" v-if="showCreateForm">
			<h3>Создать новое правило</h3>
			<form @submit.prevent="handleCreate">
				<div class="form-group">
					<label>Название:</label>
					<input v-model="newRule.title" required />
				</div>
				<div class="form-group">
					<label>Описание:</label>
					<textarea v-model="newRule.description" required></textarea>
				</div>
				<button type="submit" :disabled="loading">Создать</button>
				<button type="button" @click="showCreateForm = false">Отмена</button>
			</form>
		</div>

		<!-- Список правил -->
		<div v-if="loading" class="loading">Загрузка правил...</div>
		<div v-else-if="error" class="error">Ошибка: {{ error }}</div>
		<div v-else class="rules-list">
			<button @click="showCreateForm = true" class="create-btn">
				Создать правило
			</button>

			<div v-for="rule in rules" :key="rule.id" class="rule-item">
				<div v-if="editingRule?.id === rule.id" class="edit-form">
					<input v-model="editingRule.title" />
					<textarea v-model="editingRule.description"></textarea>
					<button @click="handleUpdate(rule.id)">Сохранить</button>
					<button @click="cancelEdit">Отмена</button>
				</div>
				<div v-else class="rule-content">
					<h3>{{ rule.title }}</h3>
					<p>{{ rule.description }}</p>
					<small>Создано: {{ formatDate(rule.createdAt) }}</small>
					<div class="actions">
						<button @click="startEdit(rule)">Редактировать</button>
						<button @click="handleDelete(rule.id)" class="delete-btn">
							Удалить
						</button>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useCommunityRules } from '@/composables/useCommunityRules'

const props = defineProps({
	communityId: {
		type: String,
		required: true,
	},
})

const { rules, loading, error, loadRules, createRule, updateRule, deleteRule } =
	useCommunityRules(props.communityId)

const showCreateForm = ref(false)
const editingRule = ref(null)
const newRule = ref({
	title: '',
	description: '',
})

const handleCreate = async () => {
	const result = await createRule(
		newRule.value.title,
		newRule.value.description
	)
	if (result.success) {
		showCreateForm.value = false
		newRule.value = { title: '', description: '' }
	} else {
		alert('Ошибка: ' + result.error)
	}
}

const startEdit = rule => {
	editingRule.value = { ...rule }
}

const cancelEdit = () => {
	editingRule.value = null
}

const handleUpdate = async ruleId => {
	const result = await updateRule(
		ruleId,
		editingRule.value.title,
		editingRule.value.description
	)
	if (result.success) {
		editingRule.value = null
	} else {
		alert('Ошибка: ' + result.error)
	}
}

const handleDelete = async ruleId => {
	if (confirm('Вы уверены, что хотите удалить это правило?')) {
		const result = await deleteRule(ruleId)
		if (!result.success) {
			alert('Ошибка: ' + result.error)
		}
	}
}

const formatDate = dateString => {
	return new Date(dateString).toLocaleDateString('ru-RU')
}

onMounted(() => {
	loadRules()
})
</script>

<style scoped>
.community-rules {
	max-width: 800px;
	margin: 0 auto;
	padding: 20px;
}

.create-form {
	background: #f5f5f5;
	padding: 20px;
	border-radius: 8px;
	margin-bottom: 20px;
}

.form-group {
	margin-bottom: 15px;
}

.form-group label {
	display: block;
	margin-bottom: 5px;
	font-weight: bold;
}

.form-group input,
.form-group textarea {
	width: 100%;
	padding: 8px;
	border: 1px solid #ddd;
	border-radius: 4px;
}

.rule-item {
	border: 1px solid #ddd;
	border-radius: 8px;
	padding: 15px;
	margin-bottom: 15px;
}

.actions {
	margin-top: 10px;
}

.actions button {
	margin-right: 10px;
}

.delete-btn {
	background: #dc3545;
	color: white;
}

.loading {
	text-align: center;
	padding: 20px;
}

.error {
	color: #dc3545;
	text-align: center;
	padding: 20px;
}
</style>
```

## 🎯 Angular

### Сервис для работы с правилами

```typescript
// services/community-rules.service.ts
import { Injectable } from '@angular/core'
import { HttpClient } from '@angular/common/http'
import { Observable, throwError } from 'rxjs'
import { map, catchError } from 'rxjs/operators'

export interface CommunityRule {
	id: string
	title: string
	description: string
	createdAt: string
	updatedAt: string
	community?: {
		id: string
		title: string
	}
}

export interface CreateRuleInput {
	communityID: string
	title: string
	description: string
}

export interface UpdateRuleInput {
	id: string
	title?: string
	description?: string
}

@Injectable({
	providedIn: 'root',
})
export class CommunityRulesService {
	private baseUrl = '/query'

	constructor(private http: HttpClient) {}

	private executeQuery<T>(query: string, variables: any = {}): Observable<T> {
		return this.http
			.post<{ data: T }>(
				this.baseUrl,
				{
					query,
					variables,
				},
				{
					withCredentials: true, // Важно для куки
				}
			)
			.pipe(
				map(response => response.data),
				catchError(error => {
					console.error('GraphQL Error:', error)
					return throwError(
						() =>
							new Error(error.error?.errors?.[0]?.message || 'Unknown error')
					)
				})
			)
	}

	getRules(communityId: string): Observable<CommunityRule[]> {
		const query = `
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
    `

		return this.executeQuery<{ communityRules: CommunityRule[] }>(query, {
			communityID: communityId,
		}).pipe(map(result => result.communityRules))
	}

	getRule(ruleId: string): Observable<CommunityRule> {
		const query = `
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
    `

		return this.executeQuery<{ communityRule: CommunityRule }>(query, {
			id: ruleId,
		}).pipe(map(result => result.communityRule))
	}

	createRule(input: CreateRuleInput): Observable<CommunityRule> {
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

		return this.executeQuery<{ createCommunityRule: CommunityRule }>(query, {
			input,
		}).pipe(map(result => result.createCommunityRule))
	}

	updateRule(input: UpdateRuleInput): Observable<CommunityRule> {
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

		return this.executeQuery<{ updateCommunityRule: CommunityRule }>(query, {
			input,
		}).pipe(map(result => result.updateCommunityRule))
	}

	deleteRule(ruleId: string): Observable<boolean> {
		const query = `
      mutation DeleteCommunityRule($id: ID!) {
        deleteCommunityRule(id: $id)
      }
    `

		return this.executeQuery<{ deleteCommunityRule: boolean }>(query, {
			id: ruleId,
		}).pipe(map(result => result.deleteCommunityRule))
	}
}
```

### Компонент Angular

```typescript
// components/community-rules.component.ts
import { Component, Input, OnInit } from '@angular/core'
import {
	CommunityRulesService,
	CommunityRule,
} from '../services/community-rules.service'

@Component({
	selector: 'app-community-rules',
	templateUrl: './community-rules.component.html',
	styleUrls: ['./community-rules.component.css'],
})
export class CommunityRulesComponent implements OnInit {
	@Input() communityId!: string

	rules: CommunityRule[] = []
	loading = false
	error: string | null = null
	showCreateForm = false
	editingRule: CommunityRule | null = null

	newRule = {
		title: '',
		description: '',
	}

	constructor(private rulesService: CommunityRulesService) {}

	ngOnInit(): void {
		this.loadRules()
	}

	loadRules(): void {
		this.loading = true
		this.error = null

		this.rulesService.getRules(this.communityId).subscribe({
			next: rules => {
				this.rules = rules
				this.loading = false
			},
			error: error => {
				this.error = error.message
				this.loading = false
			},
		})
	}

	createRule(): void {
		this.rulesService
			.createRule({
				communityID: this.communityId,
				title: this.newRule.title,
				description: this.newRule.description,
			})
			.subscribe({
				next: rule => {
					this.rules.push(rule)
					this.showCreateForm = false
					this.newRule = { title: '', description: '' }
				},
				error: error => {
					alert('Ошибка создания правила: ' + error.message)
				},
			})
	}

	startEdit(rule: CommunityRule): void {
		this.editingRule = { ...rule }
	}

	cancelEdit(): void {
		this.editingRule = null
	}

	updateRule(): void {
		if (!this.editingRule) return

		this.rulesService
			.updateRule({
				id: this.editingRule.id,
				title: this.editingRule.title,
				description: this.editingRule.description,
			})
			.subscribe({
				next: updatedRule => {
					const index = this.rules.findIndex(r => r.id === updatedRule.id)
					if (index !== -1) {
						this.rules[index] = updatedRule
					}
					this.editingRule = null
				},
				error: error => {
					alert('Ошибка обновления правила: ' + error.message)
				},
			})
	}

	deleteRule(ruleId: string): void {
		if (confirm('Вы уверены, что хотите удалить это правило?')) {
			this.rulesService.deleteRule(ruleId).subscribe({
				next: () => {
					this.rules = this.rules.filter(r => r.id !== ruleId)
				},
				error: error => {
					alert('Ошибка удаления правила: ' + error.message)
				},
			})
		}
	}

	formatDate(dateString: string): string {
		return new Date(dateString).toLocaleDateString('ru-RU')
	}
}
```

```html
<!-- community-rules.component.html -->
<div class="community-rules">
	<h2>Правила сообщества</h2>

	<!-- Форма создания -->
	<div class="create-form" *ngIf="showCreateForm">
		<h3>Создать новое правило</h3>
		<form (ngSubmit)="createRule()">
			<div class="form-group">
				<label>Название:</label>
				<input [(ngModel)]="newRule.title" name="title" required />
			</div>
			<div class="form-group">
				<label>Описание:</label>
				<textarea
					[(ngModel)]="newRule.description"
					name="description"
					required
				></textarea>
			</div>
			<button type="submit" [disabled]="loading">Создать</button>
			<button type="button" (click)="showCreateForm = false">Отмена</button>
		</form>
	</div>

	<!-- Список правил -->
	<div *ngIf="loading" class="loading">Загрузка правил...</div>
	<div *ngIf="error" class="error">Ошибка: {{ error }}</div>

	<div *ngIf="!loading && !error" class="rules-list">
		<button (click)="showCreateForm = true" class="create-btn">
			Создать правило
		</button>

		<div *ngFor="let rule of rules" class="rule-item">
			<div *ngIf="editingRule?.id === rule.id" class="edit-form">
				<input [(ngModel)]="editingRule.title" />
				<textarea [(ngModel)]="editingRule.description"></textarea>
				<button (click)="updateRule()">Сохранить</button>
				<button (click)="cancelEdit()">Отмена</button>
			</div>
			<div *ngIf="editingRule?.id !== rule.id" class="rule-content">
				<h3>{{ rule.title }}</h3>
				<p>{{ rule.description }}</p>
				<small>Создано: {{ formatDate(rule.createdAt) }}</small>
				<div class="actions">
					<button (click)="startEdit(rule)">Редактировать</button>
					<button (click)="deleteRule(rule.id)" class="delete-btn">
						Удалить
					</button>
				</div>
			</div>
		</div>
	</div>
</div>
```

## 🎯 Svelte

### Стор для работы с правилами

```javascript
// stores/communityRules.js
import { writable } from 'svelte/store'

class CommunityRulesStore {
	constructor() {
		this.rules = writable([])
		this.loading = writable(false)
		this.error = writable(null)
	}

	async loadRules(communityId) {
		this.loading.set(true)
		this.error.set(null)

		try {
			const rulesManager = new CommunityRulesManager()
			const rulesData = await rulesManager.getRules(communityId)
			this.rules.set(rulesData)
		} catch (err) {
			this.error.set(err.message)
		} finally {
			this.loading.set(false)
		}
	}

	async createRule(communityId, title, description) {
		try {
			const rulesManager = new CommunityRulesManager()
			const newRule = await rulesManager.createRule(
				communityId,
				title,
				description
			)

			this.rules.update(rules => [...rules, newRule])
			return { success: true, data: newRule }
		} catch (err) {
			return { success: false, error: err.message }
		}
	}

	async updateRule(ruleId, title, description) {
		try {
			const rulesManager = new CommunityRulesManager()
			const updatedRule = await rulesManager.updateRule(
				ruleId,
				title,
				description
			)

			this.rules.update(rules =>
				rules.map(rule => (rule.id === ruleId ? updatedRule : rule))
			)
			return { success: true, data: updatedRule }
		} catch (err) {
			return { success: false, error: err.message }
		}
	}

	async deleteRule(ruleId) {
		try {
			const rulesManager = new CommunityRulesManager()
			await rulesManager.deleteRule(ruleId)

			this.rules.update(rules => rules.filter(rule => rule.id !== ruleId))
			return { success: true }
		} catch (err) {
			return { success: false, error: err.message }
		}
	}
}

export const communityRulesStore = new CommunityRulesStore()
```

### Компонент Svelte

```svelte
<!-- CommunityRules.svelte -->
<script>
  import { onMount } from 'svelte';
  import { communityRulesStore } from '../stores/communityRules.js';

  export let communityId;

  let showCreateForm = false;
  let editingRule = null;
  let newRule = { title: '', description: '' };

  $: ({ rules, loading, error } = communityRulesStore);

  onMount(() => {
    communityRulesStore.loadRules(communityId);
  });

  async function handleCreate() {
    const result = await communityRulesStore.createRule(
      communityId,
      newRule.title,
      newRule.description
    );

    if (result.success) {
      showCreateForm = false;
      newRule = { title: '', description: '' };
    } else {
      alert('Ошибка: ' + result.error);
    }
  }

  function startEdit(rule) {
    editingRule = { ...rule };
  }

  function cancelEdit() {
    editingRule = null;
  }

  async function handleUpdate(ruleId) {
    const result = await communityRulesStore.updateRule(
      ruleId,
      editingRule.title,
      editingRule.description
    );

    if (result.success) {
      editingRule = null;
    } else {
      alert('Ошибка: ' + result.error);
    }
  }

  async function handleDelete(ruleId) {
    if (confirm('Вы уверены, что хотите удалить это правило?')) {
      const result = await communityRulesStore.deleteRule(ruleId);
      if (!result.success) {
        alert('Ошибка: ' + result.error);
      }
    }
  }

  function formatDate(dateString) {
    return new Date(dateString).toLocaleDateString('ru-RU');
  }
</script>

<div class="community-rules">
  <h2>Правила сообщества</h2>

  <!-- Форма создания -->
  {#if showCreateForm}
    <div class="create-form">
      <h3>Создать новое правило</h3>
      <form on:submit|preventDefault={handleCreate}>
        <div class="form-group">
          <label>Название:</label>
          <input bind:value={newRule.title} required />
        </div>
        <div class="form-group">
          <label>Описание:</label>
          <textarea bind:value={newRule.description} required></textarea>
        </div>
        <button type="submit" disabled={$loading}>Создать</button>
        <button type="button" on:click={() => showCreateForm = false}>Отмена</button>
      </form>
    </div>
  {/if}

  <!-- Список правил -->
  {#if $loading}
    <div class="loading">Загрузка правил...</div>
  {:else if $error}
    <div class="error">Ошибка: {$error}</div>
  {:else}
    <div class="rules-list">
      <button on:click={() => showCreateForm = true} class="create-btn">
        Создать правило
      </button>

      {#each $rules as rule (rule.id)}
        <div class="rule-item">
          {#if editingRule?.id === rule.id}
            <div class="edit-form">
              <input bind:value={editingRule.title} />
              <textarea bind:value={editingRule.description}></textarea>
              <button on:click={() => handleUpdate(rule.id)}>Сохранить</button>
              <button on:click={cancelEdit}>Отмена</button>
            </div>
          {:else}
            <div class="rule-content">
              <h3>{rule.title}</h3>
              <p>{rule.description}</p>
              <small>Создано: {formatDate(rule.createdAt)}</small>
              <div class="actions">
                <button on:click={() => startEdit(rule)}>Редактировать</button>
                <button on:click={() => handleDelete(rule.id)} class="delete-btn">
                  Удалить
                </button>
              </div>
            </div>
          {/if}
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .community-rules {
    max-width: 800px;
    margin: 0 auto;
    padding: 20px;
  }

  .create-form {
    background: #f5f5f5;
    padding: 20px;
    border-radius: 8px;
    margin-bottom: 20px;
  }

  .form-group {
    margin-bottom: 15px;
  }

  .form-group label {
    display: block;
    margin-bottom: 5px;
    font-weight: bold;
  }

  .form-group input,
  .form-group textarea {
    width: 100%;
    padding: 8px;
    border: 1px solid #ddd;
    border-radius: 4px;
  }

  .rule-item {
    border: 1px solid #ddd;
    border-radius: 8px;
    padding: 15px;
    margin-bottom: 15px;
  }

  .actions {
    margin-top: 10px;
  }

  .actions button {
    margin-right: 10px;
  }

  .delete-btn {
    background: #dc3545;
    color: white;
  }

  .loading {
    text-align: center;
    padding: 20px;
  }

  .error {
    color: #dc3545;
    text-align: center;
    padding: 20px;
  }
</style>
```

## 🎯 TypeScript типы

```typescript
// types/community-rules.ts
export interface CommunityRule {
	id: string
	communityID?: string
	title: string
	description?: string
	createdAt: string
	updatedAt: string
	community?: {
		id: string
		title: string
	}
}

export interface CreateCommunityRuleInput {
	communityID: string
	title: string
	description: string
}

export interface UpdateCommunityRuleInput {
	id: string
	title?: string
	description?: string
}

export interface CommunityRulesResponse {
	communityRules: CommunityRule[]
}

export interface CommunityRuleResponse {
	communityRule: CommunityRule
}

export interface CreateCommunityRuleResponse {
	createCommunityRule: CommunityRule
}

export interface UpdateCommunityRuleResponse {
	updateCommunityRule: CommunityRule
}

export interface DeleteCommunityRuleResponse {
	deleteCommunityRule: boolean
}

export interface GraphQLError {
	message: string
	extensions?: {
		code?: string
	}
}

export interface GraphQLResponse<T> {
	data?: T
	errors?: GraphQLError[]
}
```

## 🔧 Утилиты для валидации

```javascript
// utils/validation.js
export const validateRuleInput = input => {
	const errors = []

	if (!input.title || input.title.trim().length === 0) {
		errors.push('Название правила обязательно')
	}

	if (input.title && input.title.length > 100) {
		errors.push('Название не должно превышать 100 символов')
	}

	if (!input.description || input.description.trim().length === 0) {
		errors.push('Описание правила обязательно')
	}

	if (input.description && input.description.length > 1000) {
		errors.push('Описание не должно превышать 1000 символов')
	}

	return {
		isValid: errors.length === 0,
		errors,
	}
}

export const sanitizeRuleInput = input => {
	return {
		title: input.title?.trim() || '',
		description: input.description?.trim() || '',
	}
}
```

Эта документация предоставляет полный набор примеров для реализации функционала правил сообщества на любом современном фронтенд-фреймворке!
