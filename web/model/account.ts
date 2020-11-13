import axios from 'axios'
import Cookie from 'js-cookie'

import { updateState } from '../util/state'
import wrap from './wrap'

function setToken(token: string, expires: number) {
	Cookie.set('token', token, { expires })
	axios.defaults.headers.common.token = token
}
function delToken() {
	Cookie.remove('token')
	axios.defaults.headers.common.token = undefined
}

export function getToken() {
	return Cookie.get('token')
}

export function hasToken() {
	return Boolean(Cookie.get('token'))
}

export function login(headers: any) {
	const { remember } = headers
	const exp = remember ? 7 : undefined
	return wrap(
		axios.get('/login', { headers }),
		({ token }) => setToken(token, exp)
	)
}

export function register(data: any) {
	return wrap(
		axios.post('/register', data)
	)
}

export function getReset(user: string) {
	return wrap(
		axios.get('/reset', { params: { user } })
	)
}

export function putReset(data: any) {
	return wrap(
		axios.put(`/reset`, data)
	)
}

export function logout() {
	delToken()
	updateState({ user: undefined })
}
