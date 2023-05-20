import './index.less'
import { Layout } from 'antd'

export default function Footer() {
  return <Layout.Footer className="footer">
    <span>&copy; 2015 - {new Date().getFullYear()}</span>
    <span>&nbsp;&nbsp;</span>
    <a target="_blank" href="https://github.com/doveccl/DOJ">Doveccl Online Judge</a>
    <span>&nbsp;&nbsp;</span>
    <span>All Rights Reserved</span>
  </Layout.Footer>
}
