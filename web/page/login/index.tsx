import * as React from 'react'
import {
	Card, Form, Input, Divider,
	Checkbox, Button, Icon, message
} from 'antd'
import { FormComponentProps } from 'antd/lib/form'
import { withRouter } from 'react-router-dom'

import { login } from '../../util/account'

interface LoginFormProps {
	history: import('history').History
}

class LoginForm extends React.Component<LoginFormProps & FormComponentProps, any> {
	state = {
		loading: false
	}
	forgot = () => {
		this.props.history.push('/reset')
	}
	handleSubmit = (e: React.FormEvent) => {
		e.preventDefault()
		this.props.form.validateFields((err, values) => {
			if (!err) {
				this.setState({ loading: true })
				login(values, err => {
					this.setState({ loading: false })
					if (err) {
						message.error(err)
					} else {
						this.props.history.replace('/')
					}
				})
			}
		})
	}
	render() {
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
		return <Form onSubmit={this.handleSubmit} className="login-form">
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

const WrappedLoginForm = Form.create()(LoginForm)

interface LoginProps {
	history: import('history').History
}

class Login extends React.Component<LoginProps> {
	render() {
		return <Card title="User Login">
			<WrappedLoginForm history={this.props.history} />
		</Card>
	}
}

export default withRouter(({ history }) => <Login history={history} />)
