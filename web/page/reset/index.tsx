import React from 'react'
import { withRouter } from 'react-router-dom'

import { Card } from 'antd'

import { ResetForm } from '../../component/form/reset'
import { HistoryProps } from '../../util/interface'
import { updateState } from '../../util/state'

class Reset extends React.Component<HistoryProps> {
	public componentDidMount() {
		updateState({ path: [ 'Forgot password' ] })
	}
	public render() {
		return <Card title="Password Reset">
			<ResetForm history={this.props.history} />
		</Card>
	}
}

export default withRouter(({ history }) => <Reset history={history} />)
