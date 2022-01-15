import React, { useContext, useEffect } from 'react'
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

	return <Layout.Header style={{
		background: '#FFF',
		boxShadow: '0 1px 4px #EEE'
	}}>
		<Menu
			className="menu"
			mode="horizontal"
			key={global.user ? 'user-nemu' : 'login-menu'}
			selectedKeys={[location.pathname]}
			style={{
				border: 'none',
				justifyContent: 'end'
			}}
			onClick={({ key }) => {
				switch (key) {
					case '/login':
					case '/setting':
					case '/register':
						navigate(key)
						break
					case '/logout':
						delToken()
						navigate('/login')
						setGlobal({ user: null })
						break
				}
			}}
		>
			{global.user ? <Menu.SubMenu
				key="/user-menu" title={global.user.name}
				icon={<Avatar src={glink(global.user.mail)} />}
			>
				<Menu.Item key="/setting" icon={<IdcardOutlined />}>Setting</Menu.Item>
				<Menu.Item key="/logout" icon={<LogoutOutlined />}>Logout</Menu.Item>
			</Menu.SubMenu> : <>
				<Menu.Item key="/login" icon={<LoginOutlined />}>Login</Menu.Item>
				<Menu.Item key="/register" icon={<UserAddOutlined />}>Register</Menu.Item>
			</>}
		</Menu>
	</Layout.Header>
}
