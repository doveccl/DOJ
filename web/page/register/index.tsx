import * as React from 'react'
import { withRouter } from 'react-router-dom'

import { Card } from 'antd'

import { RegisterForm } from '../../component/form/register'
import { HistoryProps } from '../../util/interface'
import { updateState } from '../../util/state'

class Register extends React.Component<HistoryProps> {
	public componentDidMount() {
		updateState({ path: [ 'Register' ] })
	}
	public render() {
		return <Card title="User Registration">
			<RegisterForm history={this.props.history} />
		</Card>
	}
}

export default withRouter(({ history }) => <Register history={history} />)
