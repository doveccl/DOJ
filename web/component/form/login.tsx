import * as React from 'react'

import { message, Button, Checkbox, Divider, Form, Icon, Input } from 'antd'
import { FormComponentProps } from 'antd/lib/form'

import { login } from '../../model'
import { HistoryProps } from '../../util/interface'
import { updateState } from '../../util/state'

class LoginForm extends React.Component<HistoryProps & FormComponentProps> {
	public state = {
		loading: false
	}
	private forgot = () => {
		this.props.history.push('/reset')
	}
	private handleSubmit = (e: React.FormEvent) => {
		e.preventDefault()
		this.props.form.validateFields((error, values) => {
			if (!error) {
				this.setState({ loading: true })
				login(values)
					.then((user) => {
						updateState({ user })
						this.props.history.replace('/')
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
			<Form.Item label="User" {...formItemLayout}>
				{getFieldDecorator('user', {
					rules: [{ required: true, message: 'Please input your name or mail' }]
				})(
					<Input
						placeholder="Your name or mail"
						prefix={<Icon type="user" style={{ color: 'rgba(0,0,0,.25)' }} />}
					/>
				)}
			</Form.Item>
			<Form.Item label="Password" {...formItemLayout}>
				{getFieldDecorator('password', {
					rules: [{ required: true, message: 'Please input your Password' }]
				})(
					<Input
						type="password" placeholder="Your password"
						prefix={<Icon type="lock" style={{ color: 'rgba(0,0,0,.25)' }} />}
					/>
				)}
			</Form.Item>
			<Form.Item {...tailFormItemLayout}>
				{getFieldDecorator('remember', {
					valuePropName: 'checked',
					initialValue: true
				})(
					<Checkbox>Remember me for a week</Checkbox>
				)}
			</Form.Item>
			<Form.Item {...tailFormItemLayout}>
				<Button type="primary" htmlType="submit" loading={this.state.loading}>
					Login
				</Button>
				<Divider type="vertical" />
				<a onClick={this.forgot}>Forgot password</a>
			</Form.Item>
		</Form>
	}
}

export const WrappedLoginForm = Form.create()(LoginForm)
