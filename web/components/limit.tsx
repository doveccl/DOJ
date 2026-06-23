import { Button, InputNumber, Space } from 'antd'
import type { ComponentProps } from 'react'

type LimitInputProps = Omit<ComponentProps<typeof InputNumber>, 'suffix' | 'addonAfter' | 'controls'> & {
  unit: string
}

export function LimitInput({ unit, style, ...props }: LimitInputProps) {
  return (
    <Space.Compact block>
      <InputNumber {...props} controls={false} style={{ width: '100%', ...style }} />
      <Button disabled>{unit}</Button>
    </Space.Compact>
  )
}
