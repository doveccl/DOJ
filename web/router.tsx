import React from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'

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

import ManageContest from './page/manage/contest'
import ManageFile from './page/manage/file'
import ManageProblem from './page/manage/problem'
import ManageSetting from './page/manage/setting'
import ManageUser from './page/manage/user'

export const Router = () => <Routes>
	<Route index element={<Home />} />
	<Route path="/login" element={<Login />} />
	<Route path="/register" element={<Register />} />
	<Route path="/reset" element={<Reset />} />
	<Route path="/problem/:id" element={<Problem />} />
	<Route path="/problem" element={<Problems />} />
	<Route path="/contest/:id" element={<Contest />} />
	<Route path="/contest" element={<Contests />} />
	<Route path="/submission/:id" element={<Submission />} />
	<Route path="/submission" element={<Submissions />} />
	<Route path="/rank" element={<Rank />} />
	<Route path="/setting" element={<Setting />} />
	<Route path="/manage/setting" element={<ManageSetting />} />
	<Route path="/manage/user" element={<ManageUser />} />
	<Route path="/manage/problem" element={<ManageProblem />} />
	<Route path="/manage/contest" element={<ManageContest />} />
	<Route path="/manage/file" element={<ManageFile />} />
	<Route path="*" element={<Navigate to="/" replace={true} />} />
</Routes>
