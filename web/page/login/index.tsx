import * as React from 'react'
import { Card } from 'antd'
import { withRouter } from 'react-router-dom'

interface LoginProps {
	history: import('history').History
}

class Login extends React.Component<LoginProps> {
	render() {
		return <Card title="User Login">
		</Card>
	}
}

export default withRouter(({ history }) => <Login history={history} />)
