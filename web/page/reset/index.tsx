import React, { useContext, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { message, Button, Card, Form, Input, Row, Col } from 'antd'
import { getReset, putReset } from '../../model'
import { delToken } from '../../model'
import { GlobalContext } from '../../global'

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

export default function Reset() {
	const navigate = useNavigate()
	const [form] = Form.useForm()
	const [_, setGlobal] = useContext(GlobalContext)
	useEffect(() => setGlobal({ path: ['Forgot password'] }), [])

	const [loading, setLoading] = useState(false)
	const [countdown, setCountdown] = useState(0)

	return <Card title="Password Reset">
		<Form form={form} onFinish={values => {
			setLoading(true)
			putReset(values).then(() => {
				delToken()
				navigate('/login')
				setGlobal({ user: null })
				message.success('password reset success', 5)
			}).catch(e => {
				setLoading(false)
				message.error(e)
			})
		}}>
			<Form.Item label="User" {...formItemLayout}>
				<Row gutter={8}>
					<Col span={12}>
						<Form.Item name="user" noStyle>
							<Input placeholder="Your name or mail" />
						</Form.Item>
					</Col>
					<Col span={12}>
						<Button
							loading={loading}
							onClick={() => {
								setLoading(true)
								getReset(form.getFieldValue('user')).then(({ mail }) => {
									for (let i = 0; i < 60; i++) setTimeout(() => setCountdown(60 - i), 1000 * i)
									message.success(`Verify code has been sent to '${mail}'`, 5)
								}).catch(e => {
									message.error(e)
								}).finally(() => {
									setLoading(false)
								})
							}}
							disabled={!!countdown}
						>{countdown ? `(${countdown}s)` : 'Send verify code'}</Button>
					</Col>
				</Row>
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
				({ getFieldValue }) => ({
					async validator(_, value) {
						if (value !== getFieldValue('password')) {
							throw new Error('Two passwords are inconsistent')
						}
					}
				})
			]} {...formItemLayout}>
				<Input type="password" placeholder="Confirm your password" />
			</Form.Item>
			<Form.Item {...tailFormItemLayout}>
				<Button type="primary" htmlType="submit" loading={loading}>Reset</Button>
			</Form.Item>
		</Form>
	</Card>
}
