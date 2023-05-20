import { useContext, useEffect, useState } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { message, Button, Checkbox, Divider, Form, Input, Card } from 'antd'
import { UserOutlined, LockOutlined } from '@ant-design/icons'

import { login } from '../../model'
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

export default function Login() {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [_, setGlobal] = useContext(GlobalContext)
  useEffect(() => setGlobal({ path: ['Login'] }), [])

  return <Card title="User Login">
    <Form onFinish={user => {
      setLoading(true)
      login(user).then(user => {
        setGlobal({ user })
        navigate('/', { replace: true })
      }).catch(e => {
        setLoading(false)
        message.error(e)
      })
    }}>
      <Form.Item label="User" name="user" rules={[
        { required: true, message: 'Please input your name or mail' }
      ]} {...formItemLayout}>
        <Input
          placeholder="Your name or mail"
          prefix={<UserOutlined style={{ color: '#CCC' }} />}
        />
      </Form.Item>
      <Form.Item label="Password" name="password" rules={[
        { required: true, message: 'Please input your Password' }
      ]} {...formItemLayout}>
        <Input
          type="password" placeholder="Your password"
          prefix={<LockOutlined style={{ color: '#CCC' }} />}
        />
      </Form.Item>
      <Form.Item valuePropName="checked" initialValue={true} name="remember" {...tailFormItemLayout}>
        <Checkbox>Remember me for a week</Checkbox>
      </Form.Item>
      <Form.Item {...tailFormItemLayout}>
        <Button type="primary" htmlType="submit" loading={loading}>Login</Button>
        <Divider type="vertical" />
        <Link to="/reset">Forgot password</Link>
      </Form.Item>
    </Form>
  </Card>
}
