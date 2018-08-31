import axios from 'axios'
import { updateState } from './state'

function setToken(token: string) {
	axios.defaults.headers.common['token'] = token
}
function delToken() {
	axios.defaults.headers.common['token'] = ''
}

export function info() {
	const token =
		localStorage.getItem('token') ||
		sessionStorage.getItem('token')
	if (!token) { return }
	setToken(token)
	axios.get('/user/info')
		.then(({ data }) => {
			if (data.success) {
				const user = data.data
				updateState({ user })
			}
		})
		.catch(err => console.warn(err))
}

export function login(data: any, callback: (err?: string) => any) {
	const storage = data.remember ? localStorage : sessionStorage
	axios({ method: 'GET', url: '/login', headers: data })
		.then(({ data }) => {
			if (data.success) {
				const user = data.data
				storage.setItem('token', user.token)
				setToken(user.token)
				updateState({ user })
				callback()
			} else {
				callback(data.message)
			}
		})
		.catch(err => callback(err.message))
}

export function register(data: any, callback: (err?: string) => any) {
	axios.post('/register', data)
		.then(({ data }) => {
			if (data.success) {
				callback()
			} else {
				callback(data.message)
			}
		})
		.catch(err => callback(err.message))
}

export function getReset(user: string, callback: (err?: string, data?: any) => any) {
	axios.get(`/reset?user=${user}`)
		.then(({ data }) => {
			if (data.success) {
				callback(undefined, data.data)
			} else {
				callback(data.message)
			}
		})
		.catch(err => callback(err.message))
}

export function putReset(data: any, callback: (err?: string) => any) {
	axios.put(`/reset`, data)
		.then(({ data }) => {
			if (data.success) {
				callback()
			} else {
				callback(data.message)
			}
		})
		.catch(err => callback(err.message))
}

export function logout() {
	delToken()
	updateState({ user: undefined })
	localStorage.removeItem('token')
	sessionStorage.removeItem('token')
}
