import { CameraOutlined, EditOutlined } from '@ant-design/icons'
import { App as AntApp, Button, Flex, Form, Input, Modal, Tabs, Upload } from 'antd'
import type { UploadProps } from 'antd'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'

import { api, apiData, apiEmpty, uploadImage } from '../client'
import type { Me, PasswordUpdate } from '../client'
import { ProfileOverview } from '../components/profile'
import { ErrorBlock, LoadingBlock } from '../components/state'
import { useLocale } from '../locale'
import { useSession } from '../session'
import { limits } from '../utils/limits'

type AccountForm = Pick<Me, 'mail' | 'bio'>
type AccountPatch = Partial<AccountForm> & { avatar?: string }

export function UserPage() {
  const { text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const client = useQueryClient()
  const params = useParams()
  const name = params.name ?? ''
  const [solvedPage, setSolvedPage] = useState(1)
  const [solvedPageSize, setSolvedPageSize] = useState(12)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const isOwn = session.signedIn && name.toLowerCase() === session.name.toLowerCase()
  useEffect(() => {
    setSolvedPage(1)
  }, [name])
  const query = useQuery({
    queryKey: ['user', name, solvedPage, solvedPageSize],
    queryFn: () => apiData(api.GET('/api/users/{name}', { params: { path: { name }, query: { solvedPage, solvedPageSize } } })),
    enabled: name !== ''
  })
  const meQuery = useQuery({ queryKey: ['me'], queryFn: () => apiData(api.GET('/api/me')), enabled: isOwn })
  const account = useMutation({
    mutationFn: (body: AccountPatch) => apiData(api.PATCH('/api/me', { body })),
    onSuccess: (data) => {
      client.setQueryData(['me'], data)
      void client.invalidateQueries({ queryKey: ['user', name] })
      void session.refresh()
      message.success(text.common.saved)
    },
    onError: (error) => {
      message.error(error instanceof Error ? error.message : text.common.loadingFailed)
    }
  })

  if (query.isLoading || (isOwn && meQuery.isLoading)) {
    return <LoadingBlock />
  }
  if (query.isError) {
    return <ErrorBlock error={query.error} />
  }
  if (isOwn && meQuery.isError) {
    return <ErrorBlock error={meQuery.error} />
  }
  if (!query.data) {
    return <ErrorBlock error={text.common.emptyResponse} />
  }
  if (isOwn && !meQuery.data) {
    return <ErrorBlock error={text.common.emptyResponse} />
  }

  const me = meQuery.data
  const uploadAvatar: UploadProps['beforeUpload'] = (file) => {
    if (!me) {
      return Upload.LIST_IGNORE
    }
    if (!file.type.startsWith('image/')) {
      message.error(text.profile.avatarImageOnly)
      return Upload.LIST_IGNORE
    }
    void uploadImage(file)
      .then((src) => {
        account.mutate({ avatar: src })
      })
      .catch((error: unknown) => {
        message.error(error instanceof Error ? error.message : text.common.loadingFailed)
      })
    return false
  }

  return (
    <Flex vertical className="pageStack">
      <ProfileOverview
        profile={query.data}
        onSolvedPageChange={(page, pageSize) => {
          setSolvedPage(page)
          setSolvedPageSize(pageSize)
        }}
        renderAvatar={
          me
            ? (avatar) => (
                <Upload accept="image/*" beforeUpload={uploadAvatar} showUploadList={false}>
                  <button type="button" className="profileAvatarButton" aria-label={text.profile.avatar} disabled={account.isPending}>
                    {avatar}
                    <span className="profileAvatarMask">
                      <CameraOutlined />
                    </span>
                  </button>
                </Upload>
              )
            : undefined
        }
        sidebarAction={
          me ? (
            <Button block icon={<EditOutlined />} onClick={() => setSettingsOpen(true)}>
              {text.profile.editProfile}
            </Button>
          ) : undefined
        }
      />
      {me ? (
        <SettingsModal
          open={settingsOpen}
          onCancel={() => setSettingsOpen(false)}
          me={me}
          saving={account.isPending}
          onSave={(values) => account.mutate(values, { onSuccess: () => setSettingsOpen(false) })}
        />
      ) : null}
    </Flex>
  )
}

function SettingsModal({
  open,
  onCancel,
  me,
  saving,
  onSave
}: {
  open: boolean
  onCancel: () => void
  me: Me
  saving: boolean
  onSave: (values: AccountForm) => void
}) {
  const { text } = useLocale()

  return (
    <Modal open={open} destroyOnHidden title={text.profile.accountSettings} footer={null} width={560} onCancel={onCancel}>
      <Tabs
        destroyOnHidden
        items={[
          {
            key: 'account',
            label: text.profile.account,
            children: <AccountPane me={me} saving={saving} onSave={onSave} />
          },
          {
            key: 'password',
            label: text.profile.password,
            children: <PasswordPane onSaved={onCancel} />
          }
        ]}
      />
    </Modal>
  )
}

function AccountPane({ me, saving, onSave }: { me: Me; saving: boolean; onSave: (values: AccountForm) => void }) {
  const { text } = useLocale()

  return (
    <Form<AccountForm>
      layout="vertical"
      initialValues={{ mail: me.mail, bio: me.bio }}
      key={`${me.mail}:${me.bio}`}
      onFinish={onSave}
    >
      <Form.Item name="mail" label={text.profile.email} rules={[{ type: 'email' }]}>
        <Input maxLength={limits.mail} />
      </Form.Item>
      <Form.Item name="bio" label={text.profile.bio}>
        <Input.TextArea maxLength={limits.bio} showCount rows={4} />
      </Form.Item>
      <Button type="primary" htmlType="submit" loading={saving}>
        {text.profile.save}
      </Button>
    </Form>
  )
}

function PasswordPane({ onSaved }: { onSaved: () => void }) {
  const { text } = useLocale()
  const { message } = AntApp.useApp()
  const [form] = Form.useForm<PasswordUpdate>()
  const password = useMutation({
    mutationFn: (body: PasswordUpdate) => apiEmpty(api.PATCH('/api/me/password', { body })),
    onSuccess: () => {
      form.resetFields()
      message.success(text.common.saved)
      onSaved()
    },
    onError: (error) => {
      message.error(error instanceof Error ? error.message : text.common.loadingFailed)
    }
  })

  return (
    <Form<PasswordUpdate> form={form} preserve={false} layout="vertical" onFinish={(values) => password.mutate(values)}>
      <Form.Item name="oldPassword" label={text.profile.oldPassword}>
        <Input.Password />
      </Form.Item>
      <Form.Item name="newPassword" label={text.profile.newPassword}>
        <Input.Password />
      </Form.Item>
      <Button type="primary" htmlType="submit" loading={password.isPending}>
        {text.profile.save}
      </Button>
    </Form>
  )
}
