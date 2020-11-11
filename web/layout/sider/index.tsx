import * as React from 'react'
import { withRouter } from 'react-router-dom'

import { Layout, Menu } from 'antd'
import {
	HomeOutlined,
	FileTextOutlined,
	CodeOutlined,
	CalendarOutlined,
	BarcodeOutlined,
	DashboardOutlined,
	SettingOutlined,
	TeamOutlined,
	DatabaseOutlined,
	LayoutOutlined,
	FileOutlined
} from '@ant-design/icons'

import { Group } from '../../../common/interface'
import { diffGroup } from '../../../common/user'
import { HistoryProps } from '../../util/interface'
import { addListener, globalState, removeListener } from '../../util/state'

import './index.less'

class Sider extends React.Component<HistoryProps> {
	public state = { global: globalState }
	public componentDidMount() {
		addListener('sider', (global) => {
			this.setState({ global })
		})
	}
	public componentWillUnmount() {
		removeListener('sider')
	}
	public render() {
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
				selectedKeys={selectedKeys}
				defaultSelectedKeys={[ '/' ]}
				onClick={({ key }) => this.props.history.push(String(key))}
			>
				<Menu.Item key="/home">
					<HomeOutlined />
					<span>Home</span>
				</Menu.Item>
				<Menu.Item key="/problem">
					<FileTextOutlined />
					<span>Problem</span>
				</Menu.Item>
				<Menu.Item key="/contest">
					<CodeOutlined />
					<span>Contest</span>
				</Menu.Item>
				<Menu.Item key="/submission">
					<CalendarOutlined />
					<span>Submission</span>
				</Menu.Item>
				<Menu.Item key="/rank">
					<BarcodeOutlined />
					<span>Rank</span>
				</Menu.Item>
				{user && diffGroup(user, Group.admin) && <Menu.SubMenu
					key="manage"
					title={<span>
						<DashboardOutlined />
						<span>Manage</span>
					</span>}
				>
					<Menu.Item key="/manage/setting">
						<SettingOutlined />
						<span>Setting</span>
					</Menu.Item>
					<Menu.Item key="/manage/user">
						<TeamOutlined />
						<span>User</span>
					</Menu.Item>
					<Menu.Item key="/manage/problem">
						<DatabaseOutlined />
						<span>Problem</span>
					</Menu.Item>
					<Menu.Item key="/manage/contest">
						<LayoutOutlined />
						<span>Contest</span>
					</Menu.Item>
					<Menu.Item key="/manage/file">
						<FileOutlined />
						<span>File</span>
					</Menu.Item>
				</Menu.SubMenu>}
			</Menu>
		</Layout.Sider>
	}
}

export default withRouter(({ history }) => <Sider history={history} />)
