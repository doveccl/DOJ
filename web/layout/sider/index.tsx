import './index.less'
import React, { useContext } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
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
import { GlobalContext } from '../../global'

export default function Sider() {
	const location = useLocation()
	const navigate = useNavigate()
	const [global] = useContext(GlobalContext)

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
			selectedKeys={[location.pathname]}
			onClick={({ key }) => navigate(key)}
		>
			<Menu.Item key="/">
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
			{diffGroup(global.user, Group.admin) && <Menu.SubMenu
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
