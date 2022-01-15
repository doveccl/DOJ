import React from 'react'

import { message, Button, Form, Input } from 'antd'
import { FormInstance } from 'antd/lib/form'

import { putUser } from '../../model'
import { IUser } from '../../interface'

type ISettingForm = IUser & {
	password2: string
	oldPassword: string
}

interface SettingFormProps {
	user: IUser
	onUpdate: (u: IUser) => void
}

export class SettingForm extends React.Component<SettingFormProps> {
	private formRef = React.createRef<FormInstance<ISettingForm>>()
	public state = { loading: false }
	private handleSubmit = (values: IUser) => {
		this.setState({ loading: true })
		putUser(this.props.user._id, values)
			.then(() => {
				const { user, onUpdate } = this.props
				onUpdate(Object.assign(user, values))
				message.success('modify success')
				this.setState({ loading: false })
				this.formRef.current.setFieldsValue({
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

	public render() {
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
		return <Form onFinish={this.handleSubmit} initialValues={this.props.user} ref={this.formRef}>
			<Form.Item label="Name" name="name" rules={[
				{ required: true, message: 'Please input your name' },
				{ pattern: /^[a-zA-Z0-9][a-zA-Z0-9_]{2,14}$/, message: 'Invalid pattern' }
			]} {...formItemLayout}>
				<Input placeholder="Letter, number or underscore (length 3-15)" />
			</Form.Item>
			<Form.Item label="Mail" name="mail" rules={[
				{ required: true, message: 'Please input your mail' },
				{ type: 'email', message: 'Invalid email pattern' }
			]} {...formItemLayout}>
				<Input placeholder="Valid mail (for password retrieve) is required" />
			</Form.Item>
			<Form.Item label="Introduction" name="introduction" {...formItemLayout}>
				<Input.TextArea rows={5} placeholder="Your introduction" />
			</Form.Item>
			<Form.Item label="Old password" name="oldPassword" rules={[
				{ validator: async (_rule, value) => {
					if (!value && this.formRef.current.getFieldValue('password')) {
						throw new Error('Old password is required')
					}
				} }
			]} {...formItemLayout}>
				<Input type="password" placeholder="Your old password" />
			</Form.Item>
			<Form.Item label="New password" name="password" rules={[
				{ min: 6, max: 20, message: 'Length of password should be 6-20' },
				{ validator: async (_rule, value) => {
					if (value && value === this.formRef.current.getFieldValue('oldPassword')) {
						throw new Error('New password should be different to the old one')
					}
				} }
			]} {...formItemLayout}>
				<Input type="password" placeholder="Your new password (length 6-20)" />
			</Form.Item>
			<Form.Item label="Confirm" name="password2" rules={[
				{ validator: async (_rule, value) => {
					if (value !== this.formRef.current.getFieldValue('password')) {
						throw new Error('Two passwords are inconsistent')
					}
				} }
			]} {...formItemLayout}>
				<Input type="password" placeholder="Confirm your new password" />
			</Form.Item>
			<Form.Item {...tailFormItemLayout}>
				<Button type="primary" htmlType="submit" loading={this.state.loading}>
					Save
				</Button>
			</Form.Item>
		</Form>
	}
}
