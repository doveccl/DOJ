import * as React from 'react'
import * as Markdown from 'react-markdown'
import { Card, message } from 'antd'

import * as model from '../../model'
import LoginTip from '../../component/login-tip'
import { updateState, globalState } from '../../util/state'

export default class extends React.Component {
	state = {
		notification: '',
		faq: ''
	}
	componentWillMount() {
		updateState({ path: [ 'Home' ] })
		model.getConfigs()
			.then(data => {
				let state: any = {}
				for (let i of data) {
					state[i._id] = i.value
				}
				this.setState(state)
			})
			.catch(err => {
				if (globalState.user) {
					message.error(err)
				}
			})
	}
	render() {
		return <React.Fragment>
			<LoginTip />
			<Card title="Welcome to DOJ" loading={!this.state.notification}>
				<Markdown source={this.state.notification} escapeHtml={false} />
			</Card>
			<div className="divider" />
			<Card title="FAQ" loading={!this.state.faq}>
				<Markdown source={this.state.faq} escapeHtml={false} />
			</Card>
		</React.Fragment>
	}
}
