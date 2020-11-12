import * as React from 'react'
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
				mode="horizontal"
				className="menu"
				selectable={false}
				onClick={({ key }) => {
					switch (key) {
						case '/setting':
						case '/register':
							this.props.history.push(key)
							break
						default:
							logout()
							this.props.history.push('/login')
					}
				}}
			>
				{user && <Menu.SubMenu
					key="user"
					title={<span>
						<Avatar src={glink(user.mail) }/>
						<div className="hdivider" />
						<span>{user.name}</span>
					</span>}
				>
					<Menu.Item key="/setting">
						<IdcardOutlined />
						<span>Setting</span>
					</Menu.Item>
					<Menu.Item key="/logout">
						<LogoutOutlined />
						<span>Logout</span>
					</Menu.Item>
				</Menu.SubMenu>}
				{!user && <Menu.Item key="/login">
					<LoginOutlined />
					<span>Login</span>
				</Menu.Item>}
				{!user && <Menu.Item key="/register">
					<UserAddOutlined />
					<span>Register</span>
				</Menu.Item>}
			</Menu>
		</Layout.Header>
	}
}

export default withRouter(({ history }) => <Header history={history} />)
