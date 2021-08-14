import React from 'react'
import { withRouter } from 'react-router-dom'

import { Avatar, Layout, Menu } from 'antd'
import { IdcardOutlined, LogoutOutlined, LoginOutlined, UserAddOutlined } from '@ant-design/icons'

import { glink } from '../../../common/function'
import { getSelfInfo, hasToken, logout } from '../../model'
import { HistoryProps } from '../../util/interface'
import { addListener, globalState, removeListener, updateState } from '../../util/state'

import './index.less'

class Header extends React.Component<HistoryProps> {
	public state = { global: globalState }
	public componentDidMount() {
		addListener('header', (global) => {
			this.setState({ global })
		})
		if (hasToken()) {
			getSelfInfo()
				.then((user) => updateState({ user }))
				.catch(console.warn)
		}
	}
	public componentWillUnmount() {
		removeListener('header')
	}
	public render() {
		const { user } = this.state.global
		return <Layout.Header className="header">
			<Menu
				className="menu"
				mode="horizontal"
				key={user ? 'user-nemu' : 'login-menu'}
				selectedKeys={[this.props.history.location.pathname]}
				onClick={({ key }) => {
					switch (key) {
						case '/login':
						case '/setting':
						case '/register':
							this.props.history.push(key)
							break
						case '/logout':
							this.props.history.push('/login')
							logout()
							break
					}
				}}
			>
				{user ? <Menu.SubMenu
					key="/user-menu" title={user.name}
					icon={<Avatar src={glink(user.mail)} />}
				>
					<Menu.Item key="/setting" icon={<IdcardOutlined />}>Setting</Menu.Item>
					<Menu.Item key="/logout" icon={<LogoutOutlined />}>Logout</Menu.Item>
				</Menu.SubMenu> : <React.Fragment>
					<Menu.Item key="/login" icon={<LoginOutlined />}>Login</Menu.Item>
					<Menu.Item key="/register" icon={<UserAddOutlined />}>Register</Menu.Item>
				</React.Fragment>}
			</Menu>
		</Layout.Header>
	}
}

export default withRouter(({ history }) => <Header history={history} />)
