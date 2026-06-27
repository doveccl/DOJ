import { Input, Select, Table, Typography } from 'antd'
import type { TableProps } from 'antd'
import { useQuery } from '@tanstack/react-query'
import { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'

import { getProblems } from '../client'
import type { ProblemListItem, ProblemRef } from '../client'
import { useLocale } from '../locale'
import { problemCode, problemLabel } from '../utils/format'
import { limits } from '../utils/limits'
import { useDebouncedValue } from './use-debounced-value'

type Option = {
  value: number
  label: string
}

type ProblemRefInputProps = {
  value?: ProblemRef[]
  onChange?: (value: ProblemRef[]) => void
  options?: Option[]
  loading?: boolean
}

export function ProblemRefInput({ value = [], onChange, options = [], loading }: ProblemRefInputProps) {
  const { text } = useLocale()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const searchText = useDebouncedValue(search.trim())
  const remote = useQuery({
    queryKey: ['problems', 'select', searchText],
    queryFn: () => getProblems({ q: searchText }),
    enabled: open || searchText.length > 0 || value.length > 0
  })
  const remoteOptions = useMemo(() => (remote.data ?? []).map(problemOption), [remote.data])
  const [knownOptions, setKnownOptions] = useState<Option[]>([])

  useEffect(() => {
    setKnownOptions((current) => {
      const next = mergeOptions(current, options, remoteOptions)
      return sameOptions(current, next) ? current : next
    })
  }, [options, remoteOptions])

  const selectedOptions = value.map((item) => ({ value: item.id, label: optionLabel(item.id, knownOptions, options, remoteOptions) }))
  const visibleOptions = searchText ? remoteOptions : mergeOptions(options, remoteOptions)
  const selectOptions = mergeOptions(selectedOptions, visibleOptions)
  const optionMap = new Map(selectOptions.map((item) => [item.value, item.label]))

  function setIDs(ids: number[]) {
    const current = new Map(value.map((item) => [item.id, item.sort]))
    onChange?.(
      ids.map((id, index) => ({
        id,
        sort: current.get(id) || defaultProblemSort(index)
      }))
    )
  }

  function setSort(id: number, sort: string) {
    onChange?.(value.map((item) => (item.id === id ? { ...item, sort } : item)))
  }

  const columns: TableProps<ProblemRef>['columns'] = [
    {
      title: text.common.sort,
      render: (_, row) => <Input value={row.sort} maxLength={limits.sort} onChange={(event) => setSort(row.id, event.target.value)} />
    },
    {
      title: text.submissions.problem,
      render: (_, row) => (
        <Typography.Text ellipsis className="lineText">
          <Link to={`/problems/${row.id}`}>{optionMap.get(row.id) || problemCode(row.id)}</Link>
        </Typography.Text>
      )
    }
  ]

  return (
    <>
      <Select
        allowClear
        filterOption={false}
        loading={loading || remote.isFetching}
        maxTagCount="responsive"
        mode="multiple"
        onChange={setIDs}
        onOpenChange={setOpen}
        onSearch={setSearch}
        options={selectOptions}
        placeholder={text.submissions.searchProblem}
        showSearch
        style={{ width: '100%' }}
        value={value.map((item) => item.id)}
      />
      {value.length > 0 ? <Table<ProblemRef> rowKey="id" size="small" pagination={false} columns={columns} dataSource={value} /> : null}
    </>
  )
}

function problemOption(item: ProblemListItem): Option {
  return { value: item.id, label: problemLabel(item.id, item.title) }
}

function optionLabel(id: number, ...lists: Option[][]): string {
  for (const list of lists) {
    const found = list.find((item) => item.value === id)
    if (found) {
      return found.label
    }
  }
  return problemCode(id)
}

function mergeOptions(...lists: Option[][]): Option[] {
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

function sameOptions(left: Option[], right: Option[]) {
  if (left.length !== right.length) {
    return false
  }
  return left.every((item, index) => item.value === right[index].value && item.label === right[index].label)
}

export function defaultProblemSort(index: number) {
  if (index >= 0 && index < 26) {
    return String.fromCharCode('A'.charCodeAt(0) + index)
  }
  return String(index + 1)
}
