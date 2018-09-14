import * as md5 from 'md5'
import * as React from 'react'
import { withRouter } from 'react-router-dom'

import { Avatar, Icon, Layout, Menu } from 'antd'

import { getSelfInfo, hasToken, logout } from '../../model'
import { HistoryProps } from '../../util/interface'
import { addListener, globalState, removeListener, updateState } from '../../util/state'

import './index.less'

interface MenuClick { key: string }

class Header extends React.Component<HistoryProps> {
	public state = { global: globalState }
	private onClick = ({ key }: MenuClick) => {
		switch (key) {
			case '/setting':
			case '/register':
				this.props.history.push(key)
				break
			default:
				logout()
				this.props.history.push('/login')
		}
	}
	public componentWillMount() {
		addListener('header', (global) => {
			this.setState({ global })
		})
		if (!hasToken()) { return }
		getSelfInfo()
			.then((user) => updateState({ user }))
			.catch(console.warn)
	}
	public componentWillUnmount() {
		removeListener('header')
	}
	public render() {
		let { mhash, avatar } = {} as any
		const { user } = this.state.global
		if (user) {
			mhash = md5(user.mail.trim().toLowerCase())
			avatar = `//cdn.v2ex.com/gravatar/${mhash}?d=wavatar`
		}

		return <Layout.Header className="header">
			<Menu
				mode="horizontal"
				className="menu"
				selectable={false}
				onClick={this.onClick}
			>
				{user && <Menu.SubMenu
					key="user"
					title={<span>
						<Avatar className="avatar" src={avatar} />
						<span>{user.name}</span>
					</span>}
				>
					<Menu.Item key="/setting">
						<Icon type="idcard" />
						<span>Setting</span>
					</Menu.Item>
					<Menu.Item key="/logout">
						<Icon type="logout" />
						<span>Logout</span>
					</Menu.Item>
				</Menu.SubMenu>}
				{!user && <Menu.Item key="/login">
					<Icon type="login" />
					<span>Login</span>
				</Menu.Item>}
				{!user && <Menu.Item key="/register">
					<Icon type="user-add" />
					<span>Register</span>
				</Menu.Item>}
			</Menu>
		</Layout.Header>
	}
}

export default withRouter(({ history }) => <Header history={history} />)
