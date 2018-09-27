import * as React from 'react'

import { Form, Input, Select } from 'antd'
import { FormComponentProps } from 'antd/lib/form'

import { Editor } from '../../component/editor'
import { Number } from '../../component/number'
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
			<Form.Item label="Time" {...formItemLayout}>
				{getFieldDecorator('timeLimit', {
					initialValue: value ? value.timeLimit : 1.0,
					rules: [{ required: true, message: 'Please input time limit' }]
				})(
					<Number options={[
						{ name: 'ms', scale: 0.001 },
						{ name: 's', scale: 1 }
					]} default={0} />
				)}
			</Form.Item>
			<Form.Item label="Memory" {...formItemLayout}>
				{getFieldDecorator('memoryLimit', {
					initialValue: value ? value.memoryLimit : 32 * 1024 * 1024,
					rules: [{ required: true, message: 'Please input memory limit' }]
				})(
					<Number options={[
						{ name: 'Bytes', scale: 1 },
						{ name: 'KiB', scale: 1024 },
						{ name: 'MiB', scale: 1024 * 1024 },
						{ name: 'GiB', scale: 1024 * 1024 * 1024 }
					]} default={2} />
				)}
			</Form.Item>
			<Form.Item label="Data" {...formItemLayout}>
				{getFieldDecorator('data', {
					initialValue: value && value.data
				})(
					<Input placeholder="Problem data id (use Manage/File to upload data)" />
				)}
			</Form.Item>
			<Form.Item label="Content" {...formItemLayout}>
				{getFieldDecorator('content', {
					initialValue: value && value.content,
					rules: [{ required: true, message: 'Please input problem content' }]
				})(
					<Editor shortCode={true} escapeHtml={false} />
				)}
			</Form.Item>
		</Form>
	}
}

export const WrappedProblemForm = Form.create()(ProblemForm)
