import React from 'react'

import { Card } from 'antd'
import { withRouter } from 'react-router-dom'

import { SettingForm } from '../../component/form/setting'
import { HistoryProps } from '../../util/interface'
import { addListener, globalState, removeListener, updateState } from '../../util/state'

class Setting extends React.Component<HistoryProps> {
	public state = {
		global: globalState
	}
	public componentDidMount() {
		updateState({ path: [ 'Setting' ] })
		addListener('setting', (global) => {
			this.setState({ global })
		})
	}
	public componentWillUnmount() {
		removeListener('setting')
	}
	public render() {
		const { user } = this.state.global
		return <React.Fragment>
			<Card title="Setting" loading={!user}>
				{user && <SettingForm user={user} onUpdate={user => updateState({ user })} />}
			</Card>
		</React.Fragment>
	}
}

export default withRouter(({ history }) => <Setting history={history} />)
