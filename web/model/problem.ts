import axios from 'axios'
import wrap from './wrap'

export function getProblems(params?: any) {
	return wrap(
		axios.get('/problem', { params })
	)
}

export function getProblem(id: any) {
	return wrap(
		axios.get(`/problem/${id}`)
	)
}

export function postProblem(data: any) {
	return wrap(
		axios.post('/problem', data)
	)
}

export function putProblem(id: any, data: any) {
	return wrap(
		axios.put(`/problem/${id}`, data)
	)
}

export function delProblem(id: any) {
	return wrap(
		axios.delete(`/problem/${id}`)
	)
}
