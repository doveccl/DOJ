import * as React from 'react'
import { Card, Form, Input, Button, message } from 'antd'
import { FormComponentProps } from 'antd/lib/form'
import { withRouter } from 'react-router-dom'

import { getReset, putReset } from '../../util/account'

interface ResetFormProps {
	history: import('history').History
}

class ResetForm extends React.Component<ResetFormProps & FormComponentProps, any> {
	state = {
		loading: false,
		countdown: 0
	}
	handleSend = () => {
		const form = this.props.form
		const user = form.getFieldValue('user')
		this.setState({ loading: true })
		getReset(user, (err, data) => {
			if (err) {
				message.error(err)
				this.setState({ loading: false })
			} else {
				const { mail } = data
				message.success(`Verify code has been sent to '${mail}'`, 10)
				this.setState({ countdown: 60, loading: false })
				const timer = setInterval(() => {
					let countdown = this.state.countdown
					if (--countdown <= 0) {
						countdown = 0
						clearInterval(timer)
					}
					this.setState({ countdown })
				}, 1000)
			}
		})
	}
	handleSubmit = (e: React.FormEvent) => {
		e.preventDefault()
		this.props.form.validateFields((err, values) => {
			if (!err) {
				this.setState({ loading: true })
				putReset(values, err => {
					this.setState({ loading: false })
					if (err) {
						message.error(err)
					} else {
						message.success('password reset success')
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
		const { countdown } = this.state
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
			<Form.Item label="User" {...formItemLayout}>
				<Input.Group compact={true}>
					{getFieldDecorator('user')(
						<Input
							style={{ width: '70%' }}
							placeholder="Your name or mail"
						/>
					)}
					<Button
						style={{ width: '30%' }}
						loading={this.state.loading}
						onClick={this.handleSend}
						disabled={Boolean(countdown)}
					>{countdown ? `(${countdown}s)` : 'Send verify code'}</Button>
				</Input.Group>
			</Form.Item>
			<Form.Item label="Verify code" {...formItemLayout}>
				{getFieldDecorator('code', {
					rules: [{ required: true, message: 'Please input the verify code' }]
				})(
					<Input.TextArea rows={5} placeholder="Input the verify code you received" />
				)}
			</Form.Item>
			<Form.Item label="New password" {...formItemLayout}>
				{getFieldDecorator('password', {
					rules: [
						{ required: true, message: 'Please input your new password' },
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
			<Form.Item {...tailFormItemLayout}>
				<Button type="primary" htmlType="submit" loading={this.state.loading}>
					Reset
				</Button>
			</Form.Item>
		</Form>
	}
}

const WrappedResetForm = Form.create()(ResetForm)

interface ResetProps {
	history: import('history').History
}

class Reset extends React.Component<ResetProps> {
	render() {
		return <Card title="Password Reset">
			<WrappedResetForm history={this.props.history} />
		</Card>
	}
}

export default withRouter(({ history }) => <Reset history={history} />)
