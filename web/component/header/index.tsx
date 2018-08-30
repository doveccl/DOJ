import * as React from 'react'
import { Layout, Menu, Icon } from 'antd'
import { withRouter } from 'react-router-dom'

import { fetch, logout } from '../../util/account'
import { addListener, removeListener, GlobalState } from '../../util/state'

import './index.less'

interface HeaderProps {
	history: import('history').History
}

interface MenuClick {
	key: string
}

class Header extends React.Component<HeaderProps> {
	state = { global: {} as GlobalState }
	onClick = ({ key }: MenuClick) => {
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
	componentWillMount() {
		fetch() // try to load account info
		addListener('header', global => {
			this.setState({ global })
		})
	}
	componentWillUnmount() {
		removeListener('header')
	}
	render() {
		const { user } = this.state.global

		return <Layout.Header className="header">
			<Menu
				theme="light"
				mode="horizontal"
				className="menu"
				selectable={false}
				onClick={this.onClick}
			>
				{user && <Menu.SubMenu
					key="user"
					title={<span>
						<Icon type="user" />
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
					<Icon type="usergroup-add" />
					<span>Register</span>
				</Menu.Item>}
			</Menu>
		</Layout.Header>
	}
}

export default withRouter(({ history }) => <Header history={history} />)
