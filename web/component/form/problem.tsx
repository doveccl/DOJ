import * as React from 'react'

import { Form, Input, InputNumber, Select } from 'antd'
import { FormComponentProps } from 'antd/lib/form'

import Editor from '../../component/editor'
import { IProblem } from '../../util/interface'

interface ProblemFormProps extends FormComponentProps {
	value?: IProblem
}

class ProblemForm extends React.Component<ProblemFormProps> {
	public render() {
		const { value } = this.props
		const { getFieldDecorator } = this.props.form
		const formItemLayout = {
			labelCol: { xs: 24, sm: 6, md: 4 },
			wrapperCol: { xs: 24, sm: 18, md: 20 }
		}
		return <Form>
			<Form.Item label="Title" {...formItemLayout}>
				{getFieldDecorator('title', {
					initialValue: value && value.title,
					rules: [{ required: true, message: 'Please input problem title' }]
				})(
					<Input placeholder="Problem title" />
				)}
			</Form.Item>
			<Form.Item label="Tags" {...formItemLayout}>
				{getFieldDecorator('tags', {
					initialValue: value && value.tags
				})(
					<Select tokenSeparators={[',']} mode="tags" placeholder="Problem tags" />
				)}
			</Form.Item>
			<Form.Item label="Time (ms)" {...formItemLayout}>
				{getFieldDecorator('timeLimit', {
					initialValue: value ? value.timeLimit : 1000,
					rules: [{ required: true, message: 'Please input time limit' }]
				})(
					<InputNumber min={0} />
				)}
			</Form.Item>
			<Form.Item label="Memory (KiB)" {...formItemLayout}>
				{getFieldDecorator('memoryLimit', {
					initialValue: value ? value.memoryLimit : 32 * 1024,
					rules: [{ required: true, message: 'Please input memory limit' }]
				})(
					<InputNumber min={0} />
				)}
			</Form.Item>
			<Form.Item label="Content" {...formItemLayout}>
				{getFieldDecorator('content', {
					initialValue: value && value.content
				})(
					<Editor shortCode={true} escapeHtml={false} />
				)}
			</Form.Item>
		</Form>
	}
}

export default Form.create()(ProblemForm)
