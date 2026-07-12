import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'

import { useApiMessage } from './use-api-message'
import { useLocale } from '../locale'

type CrudConfig<TValues, TCreated> = {
  // Query keys to invalidate after any successful mutation.
  invalidate: readonly (readonly unknown[])[]
  create: (values: TValues) => Promise<TCreated>
  update: (id: number, values: TValues) => Promise<unknown>
  remove: (id: number) => Promise<unknown>
  // Optional side effect after a successful create (e.g. navigate to detail).
  onCreated?: (created: TCreated) => void
}

export function useEntityCrud<TValues, TCreated>(config: CrudConfig<TValues, TCreated>) {
  const { text } = useLocale()
  const client = useQueryClient()
  const { message, showError } = useApiMessage()
  const [open, setOpen] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)

  const invalidateAll = () => {
    for (const key of config.invalidate) {
      void client.invalidateQueries({ queryKey: key })
    }
  }

  function closeModal() {
    setOpen(false)
    setEditingId(null)
  }

  const create = useMutation({
    mutationFn: config.create,
    onSuccess: (created) => {
      invalidateAll()
      message.success(text.common.saved)
      closeModal()
      config.onCreated?.(created)
    },
    onError: showError
  })

  const update = useMutation({
    mutationFn: (values: TValues) => {
      if (!editingId) {
        throw new Error(text.common.emptyResponse)
      }
      return config.update(editingId, values)
    },
    onSuccess: () => {
      invalidateAll()
      message.success(text.common.saved)
      closeModal()
    },
    onError: showError
  })

  const remove = useMutation({
    mutationFn: config.remove,
    onSuccess: () => {
      invalidateAll()
      message.success(text.common.saved)
    },
    onError: showError
  })

  function openCreate() {
    setEditingId(null)
    setOpen(true)
  }

  function openEdit(id: number) {
    setEditingId(id)
    setOpen(true)
  }

  function save(values: TValues) {
    if (editingId) {
      update.mutate(values)
      return
    }
    create.mutate(values)
  }

  return {
    open,
    editingId,
    saving: create.isPending || update.isPending,
    removing: remove.isPending,
    removingId: remove.isPending ? (remove.variables as number | undefined) : undefined,
    create,
    update,
    remove,
    openCreate,
    openEdit,
    closeModal,
    save,
    showError,
    message
  }
}
