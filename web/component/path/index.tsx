import * as React from 'react'
import { Breadcrumb } from 'antd'
import { Link } from 'react-router-dom'

import { addListener, removeListener, globalState } from '../../util/state'

import './index.less'

export default class extends React.Component {
	state = { global: globalState }
	componentWillMount() {
		addListener('path', global => {
			this.setState({ global })
		})
	}
	componentWillUnmount() {
		removeListener('path')
	}
	render() {
		return <Breadcrumb className="breadcrumb">
			<Breadcrumb.Item>DOJ</Breadcrumb.Item>
			{this.state.global.path.map((i, k) => <Breadcrumb.Item key={k}>{
				typeof i === 'string' ? i : <Link to={i.url}>{i.text}</Link>
			}</Breadcrumb.Item>)}
		</Breadcrumb>
	}
}
