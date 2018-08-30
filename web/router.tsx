import * as React from 'react'
import { Switch, Route } from 'react-router-dom'

import Login from './page/login'

export default class extends React.Component {
	render() {
		return <Switch>
			<Route path="/problem" />
			<Route path="/contest" />
			<Route path="/submission" />
			<Route path="/rank" />
			<Route path="/login" component={Login} />
			<Route path="/" />
		</Switch>
	}
}
