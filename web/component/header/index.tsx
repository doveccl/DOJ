import * as md5 from 'md5'
import * as React from 'react'
import { Layout, Menu, Avatar, Icon } from 'antd'
import { withRouter } from 'react-router-dom'

import * as model from '../../model'
import * as state from '../../util/state'
import { HistoryProps } from '../../util/interface'

import './index.less'

interface MenuClick { key: string }

class Header extends React.Component<HistoryProps> {
	state = { global: state.globalState }
	onClick = ({ key }: MenuClick) => {
		switch (key) {
			case '/setting':
			case '/register':
				this.props.history.push(key)
				break
			default:
				model.logout()
				this.props.history.push('/login')
		}
	}
	componentWillMount() {
		state.addListener('header', global => {
			this.setState({ global })
		})
		if (!model.hasToken()) { return }
		model.getSelfInfo() // try to load account info
			.then(user => state.updateState({ user }))
			.catch(err => console.warn(err))
	}
	componentWillUnmount() {
		state.removeListener('header')
	}
	render() {
		let mhash, avatar
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
					<Icon type="user" />
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
