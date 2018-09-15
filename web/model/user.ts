import axios from 'axios'
import wrap from './wrap'

export function getSelfInfo() {
	return wrap(
		axios.get('/user/info')
	)
}

export function getUsers(params?: any) {
	return wrap(
		axios.get('/user', { params })
	)
}

export function putUser(id: any, data: any) {
	return wrap(
		axios.put(`/user/${id}`, data)
	)
}
