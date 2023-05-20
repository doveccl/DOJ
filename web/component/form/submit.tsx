import { useContext, useReducer, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { message, Button, Checkbox, Divider, Form, Input, Select } from 'antd'

import { Status } from '../../../common/interface'
import { Code } from '../../component/code'
import { postSubmission } from '../../model'
import { GlobalContext } from '../../global'

export function SubmitForm({ pid = '' }) {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [global] = useContext(GlobalContext)
  const [lanKey, setLanKey] = useReducer(
    (_: number, k: number) => {
      localStorage.setItem('language', `${k}`)
      return k
    },
    Number(localStorage.getItem('language'))
  )

  return <Form
    className="submit-form"
    onFinish={value => {
      postSubmission(value).then(({ _id, result }) => {
        if (result.status === Status.FREEZE) {
          setLoading(false)
          message.success('submit success')
        } else {
          navigate(`/submission/${_id}`)
        }
      }).catch(e => {
        setLoading(false)
        message.error(e)
      })
    }}
  >
    <Form.Item name="pid" initialValue={pid} hidden>
      <Input />
    </Form.Item>
    <Form.Item name="code" rules={[
      { required: true, message: 'Please input your code' }
    ]}>
      <Code options={{ maxLines: 30 }} language={global.languages?.[lanKey].suffix} />
    </Form.Item>
    <Form.Item name="language" rules={[
      { required: true, message: 'Please choose language' }
    ]} initialValue={lanKey}>
      <Select
        style={{ width: '50%', minWidth: '200px' }}
        placeholder="Choose language"
        onSelect={k => setLanKey(k)}
      >
        {global.languages?.map((lan, key) => <Select.Option
          key={key}
          value={key}
          children={lan.name}
        />)}
      </Select>
    </Form.Item>
    <Form.Item>
      <Button type="primary" htmlType="submit" loading={loading}>Submit</Button>
      <Divider type="vertical" />
      <Form.Item name="open" valuePropName="checked" initialValue={true} noStyle>
        <Checkbox>Share my code to others</Checkbox>
      </Form.Item>
    </Form.Item>
  </Form>
}
