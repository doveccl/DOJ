import { Select, Space, Typography } from 'antd'
import type { SelectProps } from 'antd'

import { useLocale } from '../locale'

const modeValues = ['default', 'strict', 'custom'] as const

type JudgeMode = (typeof modeValues)[number]

type JudgeModeOption = {
  value: JudgeMode
  label: string
  description: string
}

type JudgeModeSelectProps = Omit<SelectProps<JudgeMode, JudgeModeOption>, 'options' | 'optionRender'>

export function JudgeModeSelect(props: JudgeModeSelectProps) {
  const { text } = useLocale()
  const options = modeValues.map((value) => ({
    value,
    label: text.modes[value],
    description: text.modeDescriptions[value]
  }))

  return (
    <Select<JudgeMode, JudgeModeOption>
      popupMatchSelectWidth={360}
      {...props}
      options={options}
      optionRender={(option) => (
        <Space orientation="vertical" size={0}>
          <Typography.Text strong>{option.data.label}</Typography.Text>
          <Typography.Text type="secondary" style={{ whiteSpace: 'normal' }}>
            {option.data.description}
          </Typography.Text>
        </Space>
      )}
    />
  )
}
