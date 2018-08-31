import * as React from 'react'
import { Card } from 'antd'

import LoginTip from '../../component/login-tip'
import { updateState, globalState } from '../../util/state'

export default class extends React.Component {
	state = {
		loading: true
	}
	componentWillMount() {
		updateState({ path: [ 'Home' ] })
	}
	render() {
		return <React.Fragment>
			<LoginTip />
			<Card title="Welcome to DOJ" loading={this.state.loading}>
				// content
			</Card>
		</React.Fragment>
	}
}
