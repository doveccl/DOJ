import { Select } from 'antd'

type Option = {
  value: number
  label: string
}

export function IdSelect({
  value,
  onChange,
  options,
  loading,
  disabled
}: {
  value?: number[]
  onChange?: (value: number[]) => void
  options: Option[]
  loading?: boolean
  disabled?: boolean
}) {
  return (
    <Select
      allowClear
      disabled={disabled}
      loading={loading}
      mode="multiple"
      options={options}
      showSearch={{ optionFilterProp: 'label' }}
      style={{ width: '100%' }}
      value={value}
      onChange={onChange}
    />
  )
}
