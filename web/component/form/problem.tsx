import { useEffect } from 'react'
import { Form, FormInstance, Input, Select } from 'antd'
import { Number } from '../../component/number'
import { IProblem } from '../../interface'

const formItemLayout = {
  labelCol: { xs: 24, sm: 6, md: 4 },
  wrapperCol: { xs: 24, sm: 18, md: 20 }
}

const defaults = {
  timeLimit: 1.0,
  memoryLimit: 32 * 1024 * 1024
}

interface IProblemForm {
  problem?: IProblem
  form?: FormInstance<IProblem>
}

export function ProblemForm({ problem, form }: IProblemForm) {
  useEffect(() => form?.resetFields(), [])
  return <Form initialValues={{ ...defaults, ...problem }} form={form}>
    <Form.Item label="Title" name="title" rules={[
      { required: true, message: 'Please input problem title' }
    ]} {...formItemLayout}>
      <Input placeholder="Problem title" />
    </Form.Item>
    <Form.Item label="Tags" name="tags" {...formItemLayout}>
      <Select tokenSeparators={[',']} mode="tags" placeholder="Problem tags" />
    </Form.Item>
    <Form.Item label="Time (s)" name="timeLimit" rules={[
      { required: true, message: 'Please input time limit' }
    ]} {...formItemLayout}>
      <Number />
    </Form.Item>
    <Form.Item label="Memory (MB)" name="memoryLimit" rules={[
      { required: true, message: 'Please input memory limit' }
    ]} {...formItemLayout}>
      <Number scale={1 << 20} />
    </Form.Item>
    <Form.Item label="Data" name="data" {...formItemLayout}>
      <Input placeholder="Problem data id (use Manage/File to upload data)" />
    </Form.Item>
    <Form.Item label="Content" name="content" rules={[
      { required: true, message: 'Please input problem content' }
    ]} {...formItemLayout}>
      <Input.TextArea rows={10} />
    </Form.Item>
  </Form>
}
