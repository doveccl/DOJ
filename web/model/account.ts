import axios from 'axios'
import * as Cookie from 'js-cookie'

import wrap from './wrap'
import { updateState } from '../util/state'

function setToken(token: string, expires = 7) {
	Cookie.set('token', token, { expires })
	axios.defaults.headers.common['token'] = token
}
function delToken() {
	Cookie.remove('token')
	axios.defaults.headers.common['token'] = ''
}

export function hasToken() {
	return Boolean(Cookie.get('token'))
}

export function login(headers: any) {
	return wrap(
		axios.get('/login', { headers }),
		data => {
			setToken(data.token, headers.remember ? 7 : undefined)
		}
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
