import * as React from 'react'

import { message, Button, Card, Divider } from 'antd'

import Editor from '../../component/editor'
import { getConfig, hasToken, putConfig } from '../../model'
import { updateState } from '../../util/state'

export default class extends React.Component {
	public state = { value: undefined as string }
	private loadConfigs = () => {
		this.setState({ value: undefined })
		getConfig('notification')
			.then(({ value }) => this.setState({ value }))
			.catch(message.error)
	}
	private update = () => {
		putConfig('notification', this.state)
			.then(() => message.success('update success'))
			.catch(message.error)
	}
	public componentWillMount() {
		updateState({ path: [ 'Manage', 'Setting' ] })
		if (hasToken()) { this.loadConfigs() }
	}
	public render() {
		const { value } = this.state
		return <React.Fragment>
			<Card
				title="Set Notification"
				extra={<React.Fragment>
					<Button onClick={this.loadConfigs}>Reset</Button>
					<Divider type="vertical" />
					<Button type="primary" onClick={this.update}>Update</Button>
				</React.Fragment>}
				loading={typeof value === 'undefined'}
			>
				<Editor
					value={value}
					escapeHtml={false}
					onChange={(v) => this.setState({ value: v })}
				/>
			</Card>
		</React.Fragment>
	}
}
