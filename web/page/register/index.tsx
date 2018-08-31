import * as React from 'react'
import { Card, Form, Input, Button, message } from 'antd'
import { FormComponentProps } from 'antd/lib/form'
import { withRouter } from 'react-router-dom'

import { updateState } from '../../util/state'
import { register } from '../../util/account'

interface RegisterFormProps {
	history: import('history').History
}

class RegisterForm extends React.Component<RegisterFormProps & FormComponentProps, any> {
	state = {
		loading: false
	}
	handleSubmit = (e: React.FormEvent) => {
		e.preventDefault()
		this.props.form.validateFields((err, values) => {
			if (!err) {
				this.setState({ loading: true })
				register(values, err => {
					this.setState({ loading: false })
					if (err) {
						message.error(err)
					} else {
						message.success('registration success')
						this.props.history.replace('/login')
					}
				})
			}
		})
	}
	comparePassword = (rule: any, value: string, callback: (err?: string) => any) => {
		const form = this.props.form
		if (value && value !== form.getFieldValue('password')) {
			callback('Two passwords are inconsistent')
		} else {
			callback()
		}
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
		return <Form onSubmit={this.handleSubmit} className="register-form">
			<Form.Item label="Name" {...formItemLayout}>
				{getFieldDecorator('name', {
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
					rules: [
						{ required: true, message: 'Please input your mail' },
						{ type: 'email', message: 'Invalid email pattern' }
					]
				})(
					<Input placeholder="Valid mail (for password retrieve) is required" />
				)}
			</Form.Item>
			<Form.Item label="Password" {...formItemLayout}>
				{getFieldDecorator('password', {
					rules: [
						{ required: true, message: 'Please input your password' },
						{ min: 6, max: 20, message: 'Length of password should be 6-20' }
					]
				})(
					<Input type="password" placeholder="Your password (length 6-20)" />
				)}
			</Form.Item>
			<Form.Item label="Confirm" {...formItemLayout}>
				{getFieldDecorator('password2', {
					rules: [
						{ required: true, message: 'Please confirm your password' },
						{ validator: this.comparePassword }
					]
				})(
					<Input type="password" placeholder="Confirm your password" />
				)}
			</Form.Item>
			<Form.Item label="Invitation code" {...formItemLayout}>
				{getFieldDecorator('invitation')(
					<Input.TextArea
						rows={5}
						placeholder="This filed is not required if OJ is open for registration"
					/>
				)}
			</Form.Item>
			<Form.Item {...tailFormItemLayout}>
				<Button type="primary" htmlType="submit" loading={this.state.loading}>
					Register
				</Button>
			</Form.Item>
		</Form>
	}
}

const WrappedRegisterForm = Form.create()(RegisterForm)

interface RegisterProps {
	history: import('history').History
}

class Register extends React.Component<RegisterProps> {
	componentWillMount() {
		updateState({ path: [ 'Register' ] })
	}
	render() {
		return <Card title="User Registration">
			<WrappedRegisterForm history={this.props.history} />
		</Card>
	}
}

export default withRouter(({ history }) => <Register history={history} />)
