import * as React from 'react'
import { Layout, Menu, Icon } from 'antd'
import { withRouter } from 'react-router-dom'

import { isGroup } from '../../util/function'
import { HistoryProps } from '../../util/interface'
import { addListener, removeListener, globalState } from '../../util/state'

import './index.less'

interface MenuClick {
	key: string
}

class Sider extends React.Component<HistoryProps> {
	state = { global: globalState }
	onClick = ({ key }: MenuClick) => {
		this.props.history.push(key)
	}
	componentWillMount() {
		addListener('sider', global => {
			this.setState({ global })
		})
	}
	componentWillUnmount() {
		removeListener('sider')
	}
	render() {
		const selectedKeys: string[] = []
		const { pathname } = this.props.history.location
		let arr = pathname.match(/^(\/[^/]*)(:?\/|$)/)
		if (arr && arr[1]) { selectedKeys.push(arr[1]) }
		arr = pathname.match(/^(\/(:?manage\/)?[^/]*)(:?\/|$)/)
		if (arr && arr[1]) { selectedKeys.push(arr[1]) }

		const { user } = this.state.global

		return <Layout.Sider
			width={220}
			collapsible={true}
			breakpoint="lg"
			className="sider"
		>
			<div className="logo">
				<div>D</div>
				<h1>Online Judge</h1>
			</div>
			<Menu
				theme="dark"
				mode="inline"
				onClick={this.onClick}
				defaultSelectedKeys={[ '/' ]}
				selectedKeys={selectedKeys}
			>
				<Menu.Item key="/">
					<Icon type="home" />
					<span>Home</span>
				</Menu.Item>
				<Menu.Item key="/problem">
					<Icon type="file-text" />
					<span>Problem</span>
				</Menu.Item>
				<Menu.Item key="/contest">
					<Icon type="code" />
					<span>Contest</span>
				</Menu.Item>
				<Menu.Item key="/submission">
					<Icon type="calendar" />
					<span>Submission</span>
				</Menu.Item>
				<Menu.Item key="/rank">
					<Icon type="bar-chart" />
					<span>Rank</span>
				</Menu.Item>
				{user && isGroup(user, 'admin') && <Menu.SubMenu
					key="manage"
					title={<span>
						<Icon type="dashboard" />
						<span>Manage</span>
					</span>}
				>
					<Menu.Item key="/manage/setting">
						<Icon type="setting" />
						<span>Setting</span>
					</Menu.Item>
					<Menu.Item key="/manage/user">
						<Icon type="team" />
						<span>User</span>
					</Menu.Item>
					<Menu.Item key="/manage/problem">
						<Icon type="database" />
						<span>Problem</span>
					</Menu.Item>
					<Menu.Item key="/manage/contest">
						<Icon type="layout" />
						<span>Contest</span>
					</Menu.Item>
				</Menu.SubMenu>}
			</Menu>
		</Layout.Sider>
	}
}

export default withRouter(({ history }) => <Sider history={history} />)
