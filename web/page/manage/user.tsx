import moment from 'moment'
import { useContext, useEffect, useState } from 'react'
import { message, Button, Card, Divider, Form, Modal, Popconfirm, Table, Tag, TablePaginationConfig } from 'antd'
import { Group } from '../../../common/interface'
import { UserForm } from '../../component/form/user'
import { delUser, getUsers, inviteUser, postUser, putUser } from '../../model'
import { IUser } from '../../interface'
import { GlobalContext } from '../../global'

const renderGroup = (g: Group) => {
  switch (g) {
    case Group.admin: return <Tag>Admin</Tag>
    case Group.root: return <Tag>Root</Tag>
    default: return <Tag>Common</Tag>
  }
}

const defaultPage: TablePaginationConfig = {
  total: 0,
  current: 1,
  pageSize: 50
}

export default function ManageUser() {
  const [global, setGlobal] = useContext(GlobalContext)
  useEffect(() => setGlobal({ path: ['Manage', 'User'] }), [])

  const [form] = Form.useForm<IUser>()
  const [loading, setLoading] = useState(true)
  const [users, setUsers] = useState<IUser[]>([])
  const [open, setOpen] = useState(false)
  const [user, setUser] = useState<IUser>()
  const [pagination, setPagination] = useState(defaultPage)

  const { current, pageSize } = pagination
  useEffect(() => {
    if (global.user) {
      setLoading(true)
      getUsers({
        page: current,
        size: pageSize
      }).then(({ total, list }) => {
        setUsers(list)
        setPagination({ ...pagination, total })
      }).catch(e => {
        message.error(e)
      }).finally(() => {
        setLoading(false)
      })
    }
  }, [global.user, current, pageSize])

  return <Card
    title="Users"
    extra={<>
      <Button onClick={() => (setUser(undefined), setOpen(true))}>Create User</Button>
      <Divider type="vertical" />
      <Button
        type="primary"
        onClick={() => {
          const mail = prompt('Send invitation mail to:')
          if (!mail) return
          const hideLoad = message.loading('Sending ...', 0)
          inviteUser({ mail }).then(() => {
            message.success(`code has been send to: ${mail}`, 10)
          }).catch(e => {
            message.error(e)
          }).finally(hideLoad)
        }}
      >Invite User</Button>
    </>}
  >
    <Modal
      style={{ minWidth: 650 }}
      destroyOnClose={true}
      open={open}
      title={user ? 'Edit' : 'Create'}
      confirmLoading={loading}
      onCancel={() => setOpen(false)}
      onOk={async () => {
        setLoading(true)
        try {
          const u = await form.validateFields()
          try {
            if (user) { // Edit
              await putUser(user._id, u)
              const n = Object.assign({}, user, u)
              setUsers(users.map(o => o === user ? n : o))
            } else { // Create
              setUsers([await postUser(u), ...users])
            }
            setOpen(false)
            message.success('success')
          } catch (e) {
            message.error(`${e}`)
          }
        } catch (e) {
          // ignore form validate error
          console.debug(e)
        } finally {
          setLoading(false)
        }
      }}
    >
      <UserForm user={user} form={form} />
    </Modal>
    <Table
      rowKey="_id"
      size="middle"
      loading={loading}
      dataSource={users}
      pagination={pagination}
      onChange={setPagination}
      columns={[
        { title: 'ID', dataIndex: '_id' },
        { title: 'Name', dataIndex: 'name' },
        { title: 'Mail', dataIndex: 'mail' },
        { title: 'Group', dataIndex: 'group', render: renderGroup },
        { title: 'Join', dataIndex: 'createdAt', render: t => moment(t).format() },
        { title: 'Action', key: 'action', render: (_, r) => <>
          <a onClick={() => (setUser(r), setOpen(true))}>Edit</a>
          {global.user?._id !== r._id && <>
            <Divider type="vertical" />
            <Popconfirm title="Delete this user?" onConfirm={() => {
              delUser(r._id).then(() => {
                setUsers(users.filter(u => u._id !== r._id))
              }).catch(message.error)
            }}>
              <a style={{ color: 'red' }}>Delete</a>
            </Popconfirm>
          </>}
        </> }
      ]}
    />
  </Card>
}
