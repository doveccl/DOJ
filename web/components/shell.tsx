import {
  BookOutlined,
  CodeOutlined,
  ControlOutlined,
  DesktopOutlined,
  DownOutlined,
  FileDoneOutlined,
  LoginOutlined,
  LogoutOutlined,
  MessageOutlined,
  MoonOutlined,
  ReadOutlined,
  SendOutlined,
  SunOutlined,
  TrophyOutlined,
  TranslationOutlined,
  UserOutlined
} from '@ant-design/icons'
import { App as AntApp, Avatar, Button, Dropdown, Flex, Form, Input, Layout, Menu, Modal, Result, Space, Tabs, Typography } from 'antd'
import type { MenuProps } from 'antd'
import { useEffect, useRef, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom'

import { getSite } from '../client'
import { useColor } from '../color'
import type { ColorMode } from '../color'
import { useLocale } from '../locale'
import type { Lang } from '../locale'
import { useSession } from '../session'
import { limits } from '../utils/limits'

function navItems(text: ReturnType<typeof useLocale>['text'], admin: boolean): MenuProps['items'] {
  const items: MenuProps['items'] = [
    { key: '/problems', icon: <BookOutlined />, label: <Link to="/problems">{text.nav.problems}</Link> },
    {
      key: '/assignments',
      icon: <FileDoneOutlined />,
      label: <Link to="/assignments">{text.nav.assignments}</Link>
    },
    { key: '/contests', icon: <TrophyOutlined />, label: <Link to="/contests">{text.nav.contests}</Link> },
    { key: '/discussion', icon: <MessageOutlined />, label: <Link to="/discussion">{text.nav.discussion}</Link> },
    { key: '/rank', icon: <ReadOutlined />, label: <Link to="/rank">{text.nav.rank}</Link> },
    { key: '/submissions', icon: <SendOutlined />, label: <Link to="/submissions">{text.nav.submissions}</Link> }
  ]
  if (admin) {
    items.push({ key: '/admin', icon: <ControlOutlined />, label: <Link to="/admin">{text.nav.admin}</Link> })
  }
  return items
}

function selectedKey(pathname: string, items: MenuProps['items']) {
  const hit = items?.find((item) => item && 'key' in item && pathname.startsWith(String(item.key)))
  return hit && 'key' in hit ? [String(hit.key)] : []
}

export function Shell() {
  const location = useLocation()
  const navigate = useNavigate()
  const { lang, setLang, text } = useLocale()
  const { mode, color, setMode } = useColor()
  const session = useSession()
  const queryClient = useQueryClient()
  const previousRole = useRef(session.role)
  const [loginOpen, setLoginOpen] = useState(false)
  const site = useQuery({ queryKey: ['site'], queryFn: getSite })
  const items = navItems(text, session.admin)
  const siteName = site.data?.siteName || 'DOJ'
  const registrationOpen = site.data?.allowRegistration ?? false
  const guestBlocked = site.data?.allowGuestAccess === false && !session.signedIn

  const languageItems: MenuProps['items'] = [
    { key: 'zh', label: text.prefs.chinese },
    { key: 'en', label: text.prefs.english }
  ]
  const themeItems: MenuProps['items'] = [
    { key: 'system', icon: <DesktopOutlined />, label: text.prefs.system },
    { key: 'light', icon: <SunOutlined />, label: text.prefs.light },
    { key: 'dark', icon: <MoonOutlined />, label: text.prefs.dark }
  ]
  const profileItems: MenuProps['items'] = [
    { key: 'profile', icon: <UserOutlined />, label: <Link to={`/users/${session.name}`}>{text.prefs.profile}</Link> },
    { key: 'logout', icon: <LogoutOutlined />, label: text.prefs.logout }
  ]

  useEffect(() => {
    if (previousRole.current === session.role) {
      return
    }
    previousRole.current = session.role
    const keepSession = (query: { queryKey: readonly unknown[] }) => query.queryKey[0] !== 'me'
    void queryClient.invalidateQueries({ predicate: keepSession })
    queryClient.removeQueries({ type: 'inactive', predicate: keepSession })
  }, [queryClient, session.role])

  return (
    <>
      <Layout className="appShell">
        <Layout.Header className="appHeader">
          <Link to="/" className="brand" aria-label="DOJ home">
            <span className="brandMark">
              <CodeOutlined />
            </span>
            <Typography.Text className="brandText">{siteName}</Typography.Text>
          </Link>
          <Menu
            className="navMenu"
            mode="horizontal"
            selectedKeys={selectedKey(location.pathname, items)}
            items={items}
          />
          <Space className="userArea" size={12}>
            <Dropdown
              trigger={['hover', 'click']}
              menu={{
                items: languageItems,
                selectable: true,
                selectedKeys: [lang],
                onClick: ({ key }) => setLang(key as Lang)
              }}
            >
              <Button type="text" aria-label={text.prefs.language} icon={<TranslationOutlined />} />
            </Dropdown>
            <Dropdown
              trigger={['hover', 'click']}
              menu={{
                items: themeItems,
                selectable: true,
                selectedKeys: [mode],
                onClick: ({ key }) => setMode(key as ColorMode)
              }}
            >
              <Button
                type="text"
                aria-label={text.prefs.theme}
                icon={color === 'dark' ? <MoonOutlined /> : mode === 'system' ? <DesktopOutlined /> : <SunOutlined />}
              />
            </Dropdown>
            {session.signedIn ? (
              <Dropdown
                trigger={['hover', 'click']}
                menu={{
                  items: profileItems,
                  onClick: ({ key }) => {
                    if (key === 'logout') {
                      void session.logout().then(() => {
                        if (location.pathname.startsWith('/admin')) {
                          navigate('/')
                        }
                      })
                    }
                  }
                }}
              >
                <Button type="text" className="profileButton" aria-label={`${text.prefs.profile}: ${session.name}`}>
                  <Flex align="center" gap={8} className="profileButtonInner">
                    <Avatar size={28} src={session.avatar || undefined} icon={<UserOutlined />}>
                      {session.name.slice(0, 1).toUpperCase()}
                    </Avatar>
                    <span className="profileName">{session.name}</span>
                    <DownOutlined className="profileArrow" />
                  </Flex>
                </Button>
              </Dropdown>
            ) : (
              <Button type="primary" icon={<LoginOutlined />} onClick={() => setLoginOpen(true)}>
                {text.prefs.login}
              </Button>
            )}
          </Space>
        </Layout.Header>
        <Layout.Content className="appContent">
          {guestBlocked ? (
            <Result
              status="403"
              title={text.common.guestClosedTitle}
              subTitle={text.common.guestClosedDescription}
              extra={
                <Button type="primary" icon={<LoginOutlined />} onClick={() => setLoginOpen(true)}>
                  {text.prefs.login}
                </Button>
              }
            />
          ) : (
            <Outlet />
          )}
        </Layout.Content>
      </Layout>
      {loginOpen ? <AuthModal registrationOpen={registrationOpen} onClose={() => setLoginOpen(false)} /> : null}
    </>
  )
}

function AuthModal({ registrationOpen, onClose }: { registrationOpen: boolean; onClose: () => void }) {
  const { text } = useLocale()
  const [authTab, setAuthTab] = useState<'login' | 'register'>('login')

  useEffect(() => {
    if (!registrationOpen && authTab === 'register') {
      setAuthTab('login')
    }
  }, [authTab, registrationOpen])

  return (
    <Modal
      open
      destroyOnHidden
      title={text.prefs.loginTitle}
      footer={null}
      onCancel={onClose}
    >
      <Tabs
        activeKey={authTab}
        destroyOnHidden
        onChange={(key) => setAuthTab(key as 'login' | 'register')}
        items={[
          {
            key: 'login',
            label: text.prefs.login,
            children: <LoginPane onCancel={onClose} />
          },
          ...(registrationOpen
            ? [
                {
                  key: 'register',
                  label: text.prefs.register,
                  children: <RegisterPane onCancel={onClose} />
                }
              ]
            : [])
        ]}
      />
    </Modal>
  )
}

function LoginPane({ onCancel }: { onCancel: () => void }) {
  const { text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const [pending, setPending] = useState(false)
  const [form] = Form.useForm<{ name: string; password: string }>()

  return (
    <Form
      name="login"
      form={form}
      preserve={false}
      layout="vertical"
      onFinish={(values) => {
        async function submit() {
          setPending(true)
          try {
            await session.login(values.name.trim(), values.password)
            onCancel()
            form.resetFields()
          } catch {
            message.error(text.prefs.loginFailed)
          } finally {
            setPending(false)
          }
        }
        void submit()
      }}
    >
      <Form.Item name="name" label={text.prefs.username} rules={[{ required: true, whitespace: true }]}>
        <Input autoComplete="username" maxLength={limits.mail} />
      </Form.Item>
      <Form.Item name="password" label={text.prefs.password} rules={[{ required: true, whitespace: true }]}>
        <Input.Password autoComplete="current-password" />
      </Form.Item>
      <Flex justify="flex-end" gap={8}>
        <Button onClick={onCancel}>{text.common.cancel}</Button>
        <Button type="primary" htmlType="submit" loading={pending}>
          {text.prefs.login}
        </Button>
      </Flex>
    </Form>
  )
}

function RegisterPane({ onCancel }: { onCancel: () => void }) {
  const { text } = useLocale()
  const session = useSession()
  const { message } = AntApp.useApp()
  const [pending, setPending] = useState(false)
  const [form] = Form.useForm<{ name: string; mail: string; password: string }>()

  return (
    <Form
      name="register"
      form={form}
      preserve={false}
      layout="vertical"
      onFinish={(values) => {
        async function submit() {
          setPending(true)
          try {
            await session.register({
              name: values.name.trim(),
              mail: values.mail.trim(),
              password: values.password
            })
            onCancel()
            form.resetFields()
          } catch (error: unknown) {
            message.error(error instanceof Error ? error.message : text.prefs.loginFailed)
          } finally {
            setPending(false)
          }
        }
        void submit()
      }}
    >
      <Form.Item name="name" label={text.prefs.username} rules={[{ required: true, whitespace: true }, { min: limits.usernameMin }, { max: limits.username }]}>
        <Input autoComplete="username" maxLength={limits.username} />
      </Form.Item>
      <Form.Item name="mail" label={text.profile.email} rules={[{ required: true, type: 'email' }]}>
        <Input autoComplete="email" maxLength={limits.mail} />
      </Form.Item>
      <Form.Item name="password" label={text.prefs.password} rules={[{ required: true, min: 8 }]}>
        <Input.Password autoComplete="new-password" />
      </Form.Item>
      <Flex justify="flex-end" gap={8}>
        <Button onClick={onCancel}>{text.common.cancel}</Button>
        <Button type="primary" htmlType="submit" loading={pending}>
          {text.prefs.register}
        </Button>
      </Flex>
    </Form>
  )
}
