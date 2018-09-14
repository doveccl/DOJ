import * as React from 'react'
import { Link } from 'react-router-dom'

import { Alert } from 'antd'

import { addListener, globalState, removeListener } from '../../util/state'

import './index.less'

export default class extends React.Component {
	public state = { global: globalState }
	public componentWillMount() {
		addListener('login-tip', (global) => {
			this.setState({ global })
		})
	}
	public componentWillUnmount() {
		removeListener('login-tip')
	}
	public render() {
		return this.state.global.user ? null : <Alert
			type="warning"
			className="login-tip"
			message={<span>
				This page is not available for guest,
				please {<Link to="/login">login</Link>} first
			</span>}
		/>
	}
}
