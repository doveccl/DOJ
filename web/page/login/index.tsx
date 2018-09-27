import * as React from 'react'

import { Card } from 'antd'
import { withRouter } from 'react-router-dom'

import { WrappedLoginForm } from '../../component/form/login'
import { HistoryProps } from '../../util/interface'
import { updateState } from '../../util/state'

class Login extends React.Component<HistoryProps> {
	public componentWillMount() {
		updateState({ path: [ 'Login' ] })
	}
	public render() {
		return <Card title="User Login">
			<WrappedLoginForm history={this.props.history} />
		</Card>
	}
}

export default withRouter(({ history }) => <Login history={history} />)
