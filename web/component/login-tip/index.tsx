import * as React from 'react'
import { Alert } from 'antd'
import { Link } from 'react-router-dom'

import { addListener, removeListener, GlobalState } from '../../util/state'

import './index.less'

export default class extends React.Component {
	state = { global: {} as GlobalState }
	componentWillMount() {
		addListener('login-tip', global => {
			this.setState({ global })
		})
	}
	componentWillUnmount() {
		removeListener('login-tip')
	}
	render() {
		return this.state.global.user ? <span /> : <Alert
			type="warning"
			className="login-tip"
			message={<span>
				This page is not available for guest,
				please {<Link to="/login">login</Link>} first
			</span>}
		/>
	}
}
