import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import * as categoriesApi from '../lib/api/categories'

export const categoriesKey = ['categories'] as const

export function useCategories() {
  return useQuery({ queryKey: categoriesKey, queryFn: categoriesApi.listCategories })
}

export function useCreateCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: categoriesApi.createCategory,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: categoriesKey }),
  })
}

export function useUpdateCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: number; input: categoriesApi.UpdateCategoryInput }) =>
      categoriesApi.updateCategory(id, input),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: categoriesKey }),
  })
}

export function useDeleteCategory() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: categoriesApi.deleteCategory,
    onSuccess: () => {
      // A delete can reassign transactions to the "Other" default (see
      // server/internal/api/category_handlers.go), so the transactions
      // list/dashboard may now be stale too, not just the category list.
      void queryClient.invalidateQueries({ queryKey: categoriesKey })
      void queryClient.invalidateQueries({ queryKey: ['transactions'] })
      void queryClient.invalidateQueries({ queryKey: ['dashboard'] })
    },
  })
}
