import axios from 'axios'
import wrap from './wrap'

export function getPosts(params?: any) {
	return wrap(
		axios.get('/post', { params })
	)
}

export function postPost(data?: any) {
	return wrap(
		axios.post('/post', data)
	)
}

export function putPost(id: any, data?: any) {
	return wrap(
		axios.put(`/post/${id}`, data)
	)
}

export function delPost(id: any) {
	return wrap(
		axios.delete(`/post/${id}`)
	)
}
