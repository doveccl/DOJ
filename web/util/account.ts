import axios from 'axios'
import { updateState } from './state'

export function fetch() {
	const token =
		localStorage.getItem('token') ||
		sessionStorage.getItem('token')
	if (!token) { return }
	axios.defaults.headers.common['token'] = token
	axios.get('/api/account/info')
		.then(data => {
			console.log(data)
		})
		.catch(err => console.warn(err))
}

export function logout() {
	localStorage.removeItem('token')
	sessionStorage.removeItem('token')
	axios.defaults.headers.common['token'] = ''
}
