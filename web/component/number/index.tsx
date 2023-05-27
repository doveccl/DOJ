import { InputNumber } from 'antd'

interface NumberProps {
  scale?: number
  value?: number
  onChange?: (value: number) => void
}

export function Number(props: NumberProps) {
  const { scale = 1, value = 0, onChange } = props
  return <InputNumber
    value={value / scale}
    onChange={v => onChange?.(scale * (v ?? 0))}
  />
}
