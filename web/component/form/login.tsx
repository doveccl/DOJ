import React from 'react'

import { message, Button, Checkbox, Divider, Form, Input } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'

import { login } from '../../model'
import { HistoryProps, IUser } from '../../util/interface'
import { updateState } from '../../util/state'

export class LoginForm extends React.Component<HistoryProps> {
	public state = {
		loading: false
	}
	private forgot = () => {
		this.props.history.push('/reset')
	}
	private handleSubmit = (values: IUser) => {
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
		return <Form onFinish={this.handleSubmit}>
			<Form.Item label="User" name="user" rules={[
				{ required: true, message: 'Please input your name or mail' }
			]} {...formItemLayout}>
				<Input
					placeholder="Your name or mail"
					prefix={<UserOutlined style={{ color: 'rgba(0,0,0,.25)' }} />}
				/>
			</Form.Item>
			<Form.Item label="Password" name="password" rules={[
				{ required: true, message: 'Please input your Password' }
			]} {...formItemLayout}>
				<Input
					type="password" placeholder="Your password"
					prefix={<LockOutlined style={{ color: 'rgba(0,0,0,.25)' }} />}
				/>
			</Form.Item>
			<Form.Item valuePropName="checked" initialValue={true} name="remember" {...tailFormItemLayout}>
				<Checkbox>Remember me for a week</Checkbox>
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
