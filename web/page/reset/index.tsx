import * as React from 'react'
import { withRouter } from 'react-router-dom'

import { Card } from 'antd'

import { WrappedResetForm } from '../../component/form/reset'
import { HistoryProps } from '../../util/interface'
import { updateState } from '../../util/state'

class Reset extends React.Component<HistoryProps> {
	public componentWillMount() {
		updateState({ path: [ 'Forgot password' ] })
	}
	public render() {
		return <Card title="Password Reset">
			<WrappedResetForm history={this.props.history} />
		</Card>
	}
}

export default withRouter(({ history }) => <Reset history={history} />)
