import React from 'react'

import { InputNumber, Select, Space } from 'antd'

interface OptProps {
  name: string
  scale: number
}
interface NumberProps {
  value?: number
  onChange?: (value: number) => any
  options: OptProps[]
  default: number
}

export class Number extends React.Component<NumberProps> {
  public state = { num: 0, sel: this.props.default }
  private numChange(num: null | number) {
    const { onChange, options } = this.props
    const val = (num ?? 0) * options[this.state.sel].scale
    if (onChange) { onChange(val) }
    this.setState({ num })
  }
  private selChange(k: number) {
    const { options } = this.props
    const { num, sel } = this.state
    const val = num * options[sel].scale
    this.setState({ sel: k, num: val / options[k].scale })
  }
  public componentDidMount() {
    if (!this.props.value) return
    const { value, options } = this.props
    for (let sel = 0; sel < options.length; sel++) {
      const num = value / options[sel].scale
      if (num >= 1) { this.setState({ sel, num }) }
    }
  }
  public render() {
    return <Space.Compact>
      <InputNumber
        min={0}
        style={{ width: 150 }}
        value={this.state.num}
        onChange={this.numChange}
      />
      <Select value={this.state.sel} onChange={this.selChange}>
        {this.props.options.map((opt, i) => <Select.Option
          key={i} value={i} children={opt.name}
        />)}
      </Select>
    </Space.Compact>
  }
}
