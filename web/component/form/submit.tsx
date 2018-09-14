import * as React from 'react'

import { message, Button, Checkbox, Divider, Form, Input, Select } from 'antd'
import { FormComponentProps } from 'antd/lib/form'

import Code from '../../component/code'
import { postSubmission } from '../../model'
import { ILanguage } from '../../util/interface'

interface SubmitFormProps {
	uid: string
	pid: string
	languages: ILanguage[]
	callback: (id: any) => any
}

class SubmitForm extends React.Component<SubmitFormProps & FormComponentProps> {
	public state = {
		loading: false,
		language: undefined as string
	}
	private handleSubmit = (e: React.FormEvent) => {
		e.preventDefault()
		this.props.form.validateFields((error, values) => {
			if (!error) {
				this.setState({ loading: true })
				postSubmission(values)
					.then((data) => {
						this.setState({ loading: false })
						message.success('submit success')
						this.props.callback(data._id)
					})
					.catch((err) => {
						this.setState({ loading: false })
						message.error(err)
					})
			}
		})
	}
	public render() {
		const { uid, pid } = this.props
		const { getFieldDecorator } = this.props.form
		return <Form onSubmit={this.handleSubmit} className="submit-form">
			<Form.Item style={{ display: 'none' }}>
				{getFieldDecorator('uid', { initialValue: uid })(<Input readOnly />)}
			</Form.Item>
			<Form.Item style={{ display: 'none' }}>
				{getFieldDecorator('pid', { initialValue: pid })(<Input readOnly />)}
			</Form.Item>
			<Form.Item>
				{getFieldDecorator('code', {
					rules: [{ required: true, message: 'Please input your code' }]
				})(
					<Code options={{ maxLines: 30 }} language={this.state.language} />
				)}
			</Form.Item>
			<Form.Item>
				{getFieldDecorator('language', {
					rules: [{ required: true, message: 'Please choose language' }]
				})(
					<Select
						style={{ width: '50%', minWidth: '200px' }}
						placeholder="Choose language"
						onSelect={(value) => {
							const lan = this.props.languages[Number(value)]
							this.setState({ language: lan.suffix })
						}}
					>
						{this.props.languages.map((lan, idx) => <Select.Option
							key={idx}
							children={lan.name}
						/>)}
					</Select>
				)}
			</Form.Item>
			<Form.Item>
				<Button type="primary" htmlType="submit" loading={this.state.loading}>
					Submit
				</Button>
				<Divider type="vertical" />
				{getFieldDecorator('open', {
					valuePropName: 'checked',
					initialValue: true
				})(
					<Checkbox>Share my code to others</Checkbox>
				)}
			</Form.Item>
		</Form>
	}
}

export default Form.create()(SubmitForm)
