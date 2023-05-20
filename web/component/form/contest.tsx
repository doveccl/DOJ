import moment from 'moment'
import { useEffect } from 'react'
import { DatePicker, Form, Input, Select } from 'antd'
import { FormInstance } from 'antd/lib/form'
import { ContestType } from '../../../common/interface'
import { IContest } from '../../interface'

const formItemLayout = {
  labelCol: { xs: 24, sm: 6, md: 4 },
  wrapperCol: { xs: 24, sm: 18, md: 20 }
}

interface IContestForm {
  contest?: IContest
  form?: FormInstance<IContest>
}

function withTime(c?: IContest) {
  const startAt = c?.startAt && moment(c.startAt)
  const endAt = c?.endAt && moment(c.endAt)
  const freezeAt = c?.freezeAt && moment(c.freezeAt)
  return Object.assign({}, c, {
    startAt,
    endAt,
    freezeAt,
    time: [startAt, endAt]
  })
}

export function ContestForm({ contest, form }: IContestForm) {
  useEffect(() => form?.resetFields(), [])

  const adjust = () => {
    const type = form?.getFieldValue('type') as ContestType
    const [startAt, endAt] = form?.getFieldValue('time')
    form?.setFieldsValue({ startAt, endAt })
    if (type === ContestType.OI && startAt)
      form?.setFieldsValue({ freezeAt: startAt })
    else if (type === ContestType.ICPC && endAt)
      form?.setFieldsValue({ freezeAt: endAt.subtract(1, 'hours') })
  }

  return <Form initialValues={withTime(contest)} form={form}>
    <Form.Item label="Title" name="title" rules={[
      { required: true, message: 'Please input contest title' }
    ]} {...formItemLayout}>
      <Input placeholder="Contest title" />
    </Form.Item>
    <Form.Item label="Type" name="type" rules={[
      { required: true, message: 'Please select type' }
    ]} {...formItemLayout}>
      <Select onSelect={adjust}>
        <Select.Option value={ContestType.OI}>OI</Select.Option>
        <Select.Option value={ContestType.ICPC}>ICPC</Select.Option>
      </Select>
    </Form.Item>
    <Form.Item name="startAt" hidden>
      <DatePicker />
    </Form.Item>
    <Form.Item name="endAt" hidden>
      <DatePicker />
    </Form.Item>
    <Form.Item label="Time" name="time" rules={[
      { required: true, message: 'Please select time' }
    ]} {...formItemLayout}>
      <DatePicker.RangePicker showTime onChange={adjust} />
    </Form.Item>
    <Form.Item label="Freeze" name="freezeAt" rules={[
      { required: true, message: 'Please select freeze time' }
    ]} {...formItemLayout}>
      <DatePicker showTime />
    </Form.Item>
    <Form.Item label="Description" name="description" {...formItemLayout}>
      <Input.TextArea />
    </Form.Item>
  </Form>
}
