import * as React from 'react'

import { message, Button, Card, Divider } from 'antd'

import Editor from '../../component/editor'
import { getConfigs, hasToken, putConfigs } from '../../model'
import { updateState } from '../../util/state'

export default class extends React.Component {
	public state = {
		notification: undefined as string,
		faq: undefined as string
	}
	private loadConfigs = () => {
		this.setState({
			notification: undefined,
			faq: undefined
		})
		getConfigs()
			.then((data: any[]) => {
				data.forEach(({ _id, value }) => {
					this.setState({ [_id]: value })
				})
			})
			.catch(message.error)
	}
	private update = () => {
		putConfigs(this.state)
			.then(() => message.success('update success'))
			.catch(message.error)
	}
	public componentWillMount() {
		updateState({ path: [ 'Manage', 'Setting' ] })
		if (hasToken()) { this.loadConfigs() }
	}
	public render() {
		const { notification, faq } = this.state
		return <React.Fragment>
			<Card
				title="Set Notification &amp; FAQ"
				extra={<React.Fragment>
					<Button size="small" onClick={this.loadConfigs}>Reset</Button>
					<Divider type="vertical" />
					<Button size="small" type="primary" onClick={this.update}>Update</Button>
				</React.Fragment>}
				loading={
					typeof faq === 'undefined' ||
					typeof notification === 'undefined'
				}
			>
				<Editor
					value={notification}
					escapeHtml={false}
					onChange={(value) => this.setState({ notification: value })}
				/>
				<div className="divider" />
				<Editor
					value={faq}
					escapeHtml={false}
					onChange={(value) => this.setState({ faq: value })}
				/>
			</Card>
		</React.Fragment>
	}
}
