import * as React from 'react'

import { message, Card } from 'antd'

import LoginTip from '../../component/login-tip'
import Markdown from '../../component/markdown'
import { getConfigs, hasToken } from '../../model'
import { updateState } from '../../util/state'

export default class extends React.Component {
	public state = {
		notification: '',
		faq: ''
	}
	public componentWillMount() {
		updateState({ path: [ 'Home' ] })
		if (!hasToken()) { return }
		getConfigs()
			.then((data) => {
				const state: any = {}
				for (const i of data) {
					state[i._id] = i.value
				}
				this.setState(state)
			})
			.catch(message.error)
	}
	public render() {
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
