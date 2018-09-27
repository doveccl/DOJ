import * as React from 'react'

import { message, Button, Form, Input } from 'antd'
import { FormComponentProps } from 'antd/lib/form'

import { putUser } from '../../model'
import { IUser } from '../../util/interface'

interface SettingFormProps extends FormComponentProps {
	user: IUser
	callback: (user: IUser) => any
}

class SettingForm extends React.Component<SettingFormProps> {
	public state = {
		loading: false
	}
	private comparePassword = (rule: any, value: string, callback: (err?: string) => any) => {
		const form = this.props.form
		if (value && value !== form.getFieldValue('password')) {
			callback('Two passwords are inconsistent')
		} else {
			callback()
		}
	}
	private noSamePassword = (rule: any, value: string, callback: (err?: string) => any) => {
		const form = this.props.form
		if (value && value === form.getFieldValue('oldPassword')) {
			callback('New password should be different to the old one')
		} else {
			callback()
		}
	}
	private handleSubmit = (e: React.FormEvent) => {
		e.preventDefault()
		this.props.form.validateFields((error: any, values: any) => {
			if (!error) {
				this.setState({ loading: true })
				putUser(this.props.user._id, values)
					.then(() => {
						const { user, callback } = this.props
						callback(Object.assign(user, values))
						message.success('modify success')
						this.setState({ loading: false })
						this.props.form.setFields({
							password: '',
							password2: '',
							oldPassword: ''
						})
					})
					.catch((err) => {
						message.error(err)
						this.setState({ loading: false })
					})
			}
		})
	}
	public render() {
		const { getFieldDecorator } = this.props.form
		const formItemLayout = {
			labelCol: { xs: 24, sm: 6, md: 4 },
			wrapperCol: { xs: 24, sm: 18, md: 20 }
		}
		const tailFormItemLayout = {
			wrapperCol: {
				xs: { span: 24, offset: 0 },
				sm: { span: 18, offset: 6 },
				md: { span: 20, offset: 4 }
			}
		}
		return <Form onSubmit={this.handleSubmit}>
			<Form.Item label="Name" {...formItemLayout}>
				{getFieldDecorator('name', {
					initialValue: this.props.user.name,
					rules: [
						{ required: true, message: 'Please input your name' },
						{ pattern: /^[a-zA-Z0-9][a-zA-Z0-9_]{2,14}$/, message: 'Invalid pattern' }
					]
				})(
					<Input placeholder="Letter, number or underscore (length 3-15)" />
				)}
			</Form.Item>
			<Form.Item label="Mail" {...formItemLayout}>
				{getFieldDecorator('mail', {
					initialValue: this.props.user.mail,
					rules: [
						{ required: true, message: 'Please input your mail' },
						{ type: 'email', message: 'Invalid email pattern' }
					]
				})(
					<Input placeholder="Valid mail (for password retrieve) is required" />
				)}
			</Form.Item>
			<Form.Item label="Introduction" {...formItemLayout}>
				{getFieldDecorator('introduction', {
					initialValue: this.props.user.introduction
				})(
					<Input.TextArea
						rows={5}
						placeholder="Your introduction"
					/>
				)}
			</Form.Item>
			<Form.Item label="Old password" {...formItemLayout}>
				{getFieldDecorator('oldPassword')(
					<Input type="password" placeholder="Your old password" />
				)}
			</Form.Item>
			<Form.Item label="New password" {...formItemLayout}>
				{getFieldDecorator('password', {
					rules: [
						{ min: 6, max: 20, message: 'Length of password should be 6-20' },
						{ validator: this.noSamePassword }
					]
				})(
					<Input type="password" placeholder="Your new password (length 6-20)" />
				)}
			</Form.Item>
			<Form.Item label="Confirm" {...formItemLayout}>
				{getFieldDecorator('password2', {
					rules: [
						{ validator: this.comparePassword }
					]
				})(
					<Input type="password" placeholder="Confirm your new password" />
				)}
			</Form.Item>
			<Form.Item {...tailFormItemLayout}>
				<Button type="primary" htmlType="submit" loading={this.state.loading}>
					Save
				</Button>
			</Form.Item>
		</Form>
	}
}

export const WrappedSettingForm = Form.create()(SettingForm)
