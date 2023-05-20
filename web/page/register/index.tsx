import { useContext, useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { message, Button, Card, Form, Input } from 'antd'
import { register } from '../../model'
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

export default function Register() {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [_, setGlobal] = useContext(GlobalContext)
  useEffect(() => setGlobal({ path: ['Register'] }), [])

  return <Card title="User Registration">
    <Form onFinish={user => {
      setLoading(true)
      register(user).then(() => {
        navigate('/login')
        message.success('registration success')
      }).catch(e => {
        setLoading(false)
        message.error(e)
      })
    }}>
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
      <Form.Item label="Password" name="password" rules={[
        { required: true, message: 'Please input your password' },
        { min: 6, max: 20, message: 'Length of password should be 6-20' }
      ]} {...formItemLayout}>
        <Input type="password" placeholder="Your password (length 6-20)" />
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
      <Form.Item label="Invitation code" name="invitation" {...formItemLayout}>
        <Input.TextArea
          rows={5}
          placeholder="This filed is not required if OJ is open for registration"
        />
      </Form.Item>
      <Form.Item {...tailFormItemLayout}>
        <Button type="primary" htmlType="submit" loading={loading}>
          Register
        </Button>
      </Form.Item>
    </Form>
  </Card>
}
