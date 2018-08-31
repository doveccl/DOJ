import * as React from 'react'
import { Switch, Route } from 'react-router-dom'

import Login from './page/login'
import Register from './page/register'
import Reset from './page/reset'

import Home from './page/home'

export default class extends React.Component {
	render() {
		return <Switch>
			<Route path="/login" component={Login} />
			<Route path="/register" component={Register} />
			<Route path="/reset" component={Reset} />
			<Route path="/problem" />
			<Route path="/contest" />
			<Route path="/submission" />
			<Route path="/rank" />
			<Route path="/" component={Home} />
		</Switch>
	}
}
