import * as React from 'react'

import { message, Card } from 'antd'

import LoginTip from '../../component/login-tip'
import Markdown from '../../component/markdown'
import { getConfigs, hasToken } from '../../model'
import { updateState } from '../../util/state'

export default class extends React.Component {
	public state = {
		notification: undefined as string,
		faq: undefined as string
	}
	public componentWillMount() {
		updateState({ path: [ 'Home' ] })
		if (hasToken()) {
			getConfigs()
				.then((data: any[]) => {
					data.forEach(({ _id, value }) => {
						this.setState({ [_id]: value })
					})
				})
				.catch(message.error)
		}
	}
	public render() {
		const { notification, faq } = this.state
		return <React.Fragment>
			<LoginTip />
			<Card title="Welcome to DOJ" loading={typeof notification === 'undefined'}>
				<Markdown source={notification} escapeHtml={false} />
			</Card>
			<div className="divider" />
			<Card title="FAQ" loading={typeof faq === 'undefined'}>
				<Markdown source={faq} escapeHtml={false} />
			</Card>
		</React.Fragment>
	}
}
