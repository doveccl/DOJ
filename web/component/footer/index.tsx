import * as React from 'react'
import { Layout } from 'antd'

import './index.less'

export default class extends React.Component {
	render() {
		return <Layout.Footer className="footer">
			Copyright &copy; 2018 Doveccl Online Judge
		</Layout.Footer>
	}
}
