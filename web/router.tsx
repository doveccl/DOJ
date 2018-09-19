import * as React from 'react'
import { Redirect, Route, Switch } from 'react-router-dom'

import Contests from './page/contest'
import Contest from './page/contest/id'
import Home from './page/home'
import Login from './page/login'
import Problems from './page/problem'
import Problem from './page/problem/id'
import Rank from './page/rank'
import Register from './page/register'
import Reset from './page/reset'
import Setting from './page/setting'
import Submissions from './page/submission'
import Submission from './page/submission/id'

import ManageProblem from './page/manage/problem'
import ManageSetting from './page/manage/setting'
import ManageUser from './page/manage/user'

export default () => <Switch>
	<Route path="/home" component={Home} />
	<Route path="/login" component={Login} />
	<Route path="/register" component={Register} />
	<Route path="/reset" component={Reset} />
	<Route path="/problem/:id" component={Problem} />
	<Route path="/problem" component={Problems} />
	<Route path="/contest/:id" component={Contest} />
	<Route path="/contest" component={Contests} />
	<Route path="/submission/:id" component={Submission} />
	<Route path="/submission" component={Submissions} />
	<Route path="/rank" component={Rank} />
	<Route path="/setting" component={Setting} />
	<Route path="/manage/setting" component={ManageSetting} />
	<Route path="/manage/user" component={ManageUser} />
	<Route path="/manage/problem" component={ManageProblem} />
	<Redirect to="/home" />
</Switch>
