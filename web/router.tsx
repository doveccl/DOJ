import * as React from 'react'
import { Route, Switch } from 'react-router-dom'

import Contests from './page/contest'
import Contest from './page/contest/id'
import Home from './page/home'
import Login from './page/login'
import Problems from './page/problem'
import Problem from './page/problem/id'
import Register from './page/register'
import Reset from './page/reset'

export default class extends React.Component {
	public render() {
		return <Switch>
			<Route path="/login" component={Login} />
			<Route path="/register" component={Register} />
			<Route path="/reset" component={Reset} />
			<Route path="/problem/:id" component={Problem} />
			<Route path="/problem" component={Problems} />
			<Route path="/contest/:id" component={Contest} />
			<Route path="/contest" component={Contests} />
			<Route path="/submission" />
			<Route path="/rank" />
			<Route path="/" component={Home} />
		</Switch>
	}
}
