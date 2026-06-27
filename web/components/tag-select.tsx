import { Select } from 'antd'
import type { CSSProperties } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useMemo } from 'react'

import { getTags } from '../client'
import { useRemoteSearch } from './use-debounced-value'

type TagSelectProps = {
  kind: 'problem' | 'discussion'
  value?: string | string[]
  onChange?: (value: string | string[]) => void
  mode?: 'tags'
  placeholder?: string
  allowClear?: boolean
  style?: CSSProperties
}

export function TagSelect({ kind, value, onChange, mode, placeholder, allowClear, style }: TagSelectProps) {
  const remote = useRemoteSearch()
  const query = useQuery({
    queryKey: ['tags', kind, remote.searchText],
    queryFn: () => getTags(kind, remote.searchText),
    enabled: remote.active
  })
  const options = useMemo(() => tagOptions(value, query.data ?? []), [query.data, value])

  return (
    <Select
      allowClear={allowClear}
      loading={query.isFetching}
      maxTagCount={mode ? 'responsive' : undefined}
      mode={mode}
      onChange={onChange}
      onOpenChange={remote.setOpen}
      onSearch={remote.setSearch}
      options={options}
      placeholder={placeholder}
      showSearch={{ filterOption: false }}
      style={style}
      tokenSeparators={mode ? [',', '，', ' '] : undefined}
      value={value}
    />
  )
}

function tagOptions(value: string | string[] | undefined, remote: string[]) {
  const selected = Array.isArray(value) ? value : value ? [value] : []
  const merged = new Set([...selected, ...remote])
  return Array.from(merged, (tag) => ({ value: tag, label: tag }))
}
