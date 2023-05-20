import './index.less'
import { useContext } from 'react'
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

  return <Layout.Sider width={220} collapsible={true} breakpoint="lg" className="sider">
    <div className="logo">
      <div>D</div>
      <h1>Online Judge</h1>
    </div>
    <Menu
      theme="dark"
      mode="inline"
      selectedKeys={[location.pathname]}
      onClick={({ key }) => navigate(key)}
      items={[
        { key: '/', label: 'Home', icon: <HomeOutlined /> },
        { key: '/problem', label: 'Problem', icon: <FileTextOutlined /> },
        { key: '/contest', label: 'Contest', icon: <CodeOutlined /> },
        { key: '/submission', label: 'Submission', icon: <CalendarOutlined /> },
        { key: '/rank', label: 'Rank', icon: <BarcodeOutlined /> },
        diffGroup(global.user, Group.admin) ? {
          key: 'manage',
          label: 'Manage',
          icon: <DashboardOutlined />,
          children: [
            { key: '/manage/setting', label: 'Setting', icon: <SettingOutlined /> },
            { key: '/manage/user', label: 'User', icon: <TeamOutlined /> },
            { key: '/manage/problem', label: 'Problem', icon: <DatabaseOutlined /> },
            { key: '/manage/contest', label: 'Contest', icon: <LayoutOutlined /> },
            { key: '/manage/file', label: 'File', icon: <FileOutlined /> },
          ]
        } : null
      ]}
    />
  </Layout.Sider>
}
