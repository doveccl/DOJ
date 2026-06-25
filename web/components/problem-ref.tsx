import { Input, Select, Table, Typography } from 'antd'
import type { TableProps } from 'antd'
import { Link } from 'react-router-dom'

import type { ProblemRef } from '../client'
import { useLocale } from '../locale'
import { problemCode } from '../utils/format'
import { limits } from '../utils/limits'

type Option = {
  value: number
  label: string
}

type ProblemRefInputProps = {
  value?: ProblemRef[]
  onChange?: (value: ProblemRef[]) => void
  options: Option[]
  loading?: boolean
}

export function ProblemRefInput({ value = [], onChange, options, loading }: ProblemRefInputProps) {
  const { text } = useLocale()
  const optionMap = new Map(options.map((item) => [item.value, item.label]))

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
      <Select mode="multiple" value={value.map((item) => item.id)} options={options} loading={loading} onChange={setIDs} />
      {value.length > 0 ? <Table<ProblemRef> rowKey="id" size="small" pagination={false} columns={columns} dataSource={value} /> : null}
    </>
  )
}

export function defaultProblemSort(index: number) {
  if (index >= 0 && index < 26) {
    return String.fromCharCode('A'.charCodeAt(0) + index)
  }
  return String(index + 1)
}
