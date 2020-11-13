import React from 'react'

import { message, Button, Form, Input } from 'antd'
import { FormInstance } from 'antd/lib/form'

import { getReset, putReset } from '../../model'
import { logout } from '../../model'
import { HistoryProps } from '../../util/interface'

export class ResetForm extends React.Component<HistoryProps> {
	private formRef = React.createRef<FormInstance<any>>()
	public state = {
		loading: false,
		countdown: 0
	}
	private handleSend() {
		const user = this.formRef.current.getFieldValue('user')
		this.setState({ loading: true })
		getReset(user)
			.then(({ mail }) => {
				message.success(`Verify code has been sent to '${mail}'`, 5)
				this.setState({ countdown: 60, loading: false })
				const timer = setInterval(() => {
					let countdown = this.state.countdown
					if (--countdown <= 0) {
						countdown = 0
						clearInterval(timer)
					}
					this.setState({ countdown })
				}, 1000)
			})
			.catch((err) => {
				message.error(err)
				this.setState({ loading: false })
			})
	}
	private handleSubmit(values: any) {
		this.setState({ loading: true })
		putReset(values)
			.then(() => {
				logout()
				message.success('password reset success')
				this.props.history.replace('/login')
			})
			.catch((err) => {
				message.error(err)
				this.setState({ loading: false })
			})
	}
	public render() {
		const { countdown } = this.state
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
		return <Form onFinish={this.handleSubmit} ref={this.formRef}>
			<Form.Item {...formItemLayout}>
				<Input.Group compact={true}>
					<Form.Item label="User" name="user" noStyle>
						<Input
							style={{ width: '70%' }}
							placeholder="Your name or mail"
						/>
					</Form.Item>
					<Button
						style={{ width: '30%' }}
						loading={this.state.loading}
						onClick={this.handleSend}
						disabled={Boolean(countdown)}
					>{countdown ? `(${countdown}s)` : 'Send verify code'}</Button>
				</Input.Group>
			</Form.Item>
			<Form.Item label="Verify code" name="code" rules={[
				{ required: true, message: 'Please input the verify code' }
			]} {...formItemLayout}>
				<Input.TextArea rows={5} placeholder="Input the verify code you received" />
			</Form.Item>
			<Form.Item label="New password" name="password" rules={[
				{ required: true, message: 'Please input your new password' },
				{ min: 6, max: 20, message: 'Length of password should be 6-20' }
			]} {...formItemLayout}>
				<Input type="password" placeholder="Your new password (length 6-20)" />
			</Form.Item>
			<Form.Item label="Confirm" name="password2" rules={[
				{ required: true, message: 'Please confirm your password' },
				{ validator: async (_rule, value) => {
					if (value !== this.formRef.current.getFieldValue('password')) {
						throw new Error('Two passwords are inconsistent')
					}
				} }
			]} {...formItemLayout}>
				<Input type="password" placeholder="Confirm your password" />
			</Form.Item>
			<Form.Item {...tailFormItemLayout}>
				<Button type="primary" htmlType="submit" loading={this.state.loading}>
					Reset
				</Button>
			</Form.Item>
		</Form>
	}
}
