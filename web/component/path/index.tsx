import React from 'react'
import { Link } from 'react-router-dom'

import { Breadcrumb } from 'antd'

import { addListener, globalState, removeListener } from '../../util/state'

import './index.less'

export class Path extends React.Component {
	public state = { global: globalState }
	public componentDidMount() {
		addListener('path', (global) => {
			this.setState({ global })
		})
	}
	public componentWillUnmount() {
		removeListener('path')
	}
	public render() {
		return <Breadcrumb className="breadcrumb">
			<Breadcrumb.Item>DOJ</Breadcrumb.Item>
			{this.state.global.path.map((i, k) => <Breadcrumb.Item key={k}>{
				typeof i === 'string' ? i : <Link to={i.url}>{i.text}</Link>
			}</Breadcrumb.Item>)}
		</Breadcrumb>
	}
}
