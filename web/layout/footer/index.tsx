import React from 'react'

import { Layout } from 'antd'

import './index.less'

export default class extends React.Component {
	public render() {
		return <Layout.Footer className="footer">
			<span>Copyright &copy; 2018 </span>
			<a target="_blank" href="https://github.com/doveccl/DOJ">Doveccl Online Judge</a>
		</Layout.Footer>
	}
}
