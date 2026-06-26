import { Select } from 'antd'
import type { CSSProperties } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useMemo, useState } from 'react'

import { getTags } from '../client'
import { useDebouncedValue } from './use-debounced-value'

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
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const searchText = useDebouncedValue(search.trim())
  const query = useQuery({
    queryKey: ['tags', kind, searchText],
    queryFn: () => getTags(kind, searchText),
    enabled: open || searchText.length > 0
  })
  const options = useMemo(() => tagOptions(value, query.data ?? []), [query.data, value])

  return (
    <Select
      allowClear={allowClear}
      filterOption={false}
      loading={query.isFetching}
      maxTagCount={mode ? 'responsive' : undefined}
      mode={mode}
      onChange={onChange}
      onOpenChange={setOpen}
      onSearch={setSearch}
      options={options}
      placeholder={placeholder}
      showSearch
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
