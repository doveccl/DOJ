import * as React from 'react'

import { message, Card } from 'antd'

import { LoginTip } from '../../component/login-tip'
import { MarkDown } from '../../component/markdown'
import { getConfig, hasToken } from '../../model'
import { updateState } from '../../util/state'

export default class extends React.Component {
	public state = { value: undefined as string }
	public componentWillMount() {
		updateState({ path: [ 'Home' ] })
		if (hasToken()) {
			getConfig('notification')
				.then(({ value }) => this.setState({ value }))
				.catch(message.error)
		}
	}
	public render() {
		const { value } = this.state
		return <React.Fragment>
			<LoginTip />
			<Card title="Welcome to DOJ" loading={typeof value === 'undefined'}>
				<MarkDown source={value} escapeHtml={false} />
			</Card>
		</React.Fragment>
	}
}
