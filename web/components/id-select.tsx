import { Select } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { useMemo, useState } from 'react'

import { getAdminMembers } from '../client'
import { useDebouncedValue } from './use-debounced-value'

type Option = {
  value: number
  label: string
}

export function IdSelect({
  value,
  onChange,
  options,
  loading,
  disabled,
  kind
}: {
  value?: number[]
  onChange?: (value: number[]) => void
  options?: Option[]
  loading?: boolean
  disabled?: boolean
  kind?: 'users' | 'groups'
}) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const searchText = useDebouncedValue(search.trim())
  const selected = (value ?? []).join(',')
  const remote = useQuery({
    queryKey: ['admin-members', kind, searchText, selected],
    queryFn: () =>
      getAdminMembers({
        q: searchText,
        users: kind === 'users' ? selected : undefined,
        groups: kind === 'groups' ? selected : undefined
      }),
    enabled: Boolean(kind) && (open || searchText.length > 0 || selected.length > 0)
  })
  const remoteOptions = useMemo(() => {
    if (!kind) {
      return []
    }
    const rows = kind === 'users' ? (remote.data?.users ?? []) : (remote.data?.groups ?? [])
    return rows.map((item) => ({ value: item.id, label: item.name }))
  }, [kind, remote.data])
  const selectedItems = selectedOptions(value ?? [], options ?? [], remoteOptions)
  const visibleOptions = kind ? (searchText ? remoteOptions : mergeOptions(options ?? [], remoteOptions)) : (options ?? [])
  const mergedOptions = mergeOptions(selectedItems, visibleOptions)
  const showSearch = kind ? { filterOption: false } : { optionFilterProp: 'label' }

  return (
    <Select
      allowClear
      disabled={disabled}
      loading={loading || remote.isFetching}
      maxTagCount="responsive"
      mode="multiple"
      onOpenChange={setOpen}
      onSearch={kind ? setSearch : undefined}
      options={mergedOptions}
      showSearch={showSearch}
      style={{ width: '100%' }}
      value={value}
      onChange={onChange}
    />
  )
}

function selectedOptions(values: number[], ...lists: Option[][]) {
  return values.map((value) => ({ value, label: optionLabel(value, lists) }))
}

function optionLabel(value: number, lists: Option[][]) {
  for (const list of lists) {
    const found = list.find((item) => item.value === value)
    if (found) {
      return found.label
    }
  }
  return String(value)
}

function mergeOptions(...lists: Option[][]) {
  const merged = new Map<number, string>()
  for (const list of lists) {
    for (const item of list) {
      if (!merged.has(item.value)) {
        merged.set(item.value, item.label)
      }
    }
  }
  return Array.from(merged, ([value, label]) => ({ value, label }))
}
