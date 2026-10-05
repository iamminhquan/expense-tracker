import { api } from './client'
import type { CategoriesResponse, Category } from './types'

export function listCategories() {
  return api.get<CategoriesResponse>('/api/categories')
}

export interface CreateCategoryInput {
  name: string
  type: 'expense' | 'income'
  color: string
}

export function createCategory(input: CreateCategoryInput) {
  return api.post<Category>('/api/categories', input)
}

export interface UpdateCategoryInput {
  name?: string
  color?: string
}

export function updateCategory(id: number, input: UpdateCategoryInput) {
  return api.patch<Category>(`/api/categories/${id}`, input)
}

export function deleteCategory(id: number) {
  return api.delete<void>(`/api/categories/${id}`)
}
