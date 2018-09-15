import * as React from 'react'

import { Card } from 'antd'
import { withRouter } from 'react-router-dom'

import WrappedSettingForm from '../../component/form/setting'
import { HistoryProps } from '../../util/interface'
import { addListener, globalState, removeListener, updateState } from '../../util/state'

class Setting extends React.Component<HistoryProps> {
	public state = {
		global: globalState
	}
	public componentWillMount() {
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
				{user && <WrappedSettingForm
					user={user}
					callback={(u) => updateState({ user: u })}
				/>}
			</Card>
		</React.Fragment>
	}
}

export default withRouter(({ history }) => <Setting history={history} />)
