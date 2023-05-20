import { useContext, useEffect } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Avatar, Layout, Menu } from 'antd'
import { IdcardOutlined, LogoutOutlined, LoginOutlined, UserAddOutlined } from '@ant-design/icons'

import { glink } from '../../../common/function'
import { getSelfInfo, hasToken, delToken } from '../../model'
import { GlobalContext } from '../../global'

export default function Header() {
  const location = useLocation()
  const navigate = useNavigate()
  const [global, setGlobal] = useContext(GlobalContext)

  useEffect(() => {
    hasToken() && getSelfInfo()
      .then(user => setGlobal({ user }))
      .catch(e => console.warn('auto login fail', e))
  }, [])

  function logout() {
    delToken()
    navigate('/login')
    setGlobal({ user: undefined })
  }

  return <Layout.Header style={{ background: '#FFF', boxShadow: '0 1px 4px #EEE' }}>
    <Menu
      className="menu"
      mode="horizontal"
      key={global.user ? 'user-nemu' : 'login-menu'}
      selectedKeys={[location.pathname]}
      style={{ border: 'none', justifyContent: 'end' }}
      onClick={e => e.key === 'logout' ? logout() : navigate(e.key)}
      items={global.user ? [{
        key: 'user',
        label: global.user.name,
        icon: <Avatar style={{ verticalAlign: 'middle' }} src={glink(global.user.mail)} />,
        children: [
          { key: '/setting', label: 'Setting', icon: <IdcardOutlined /> },
          { key: 'logout', label: 'Logout', icon: <LogoutOutlined /> }
        ]
      }] : [
        { key: '/login', label: 'Login', icon: <LoginOutlined /> },
        { key: '/register', label: 'Register', icon: <UserAddOutlined /> }
      ]}
    />
  </Layout.Header>
}
