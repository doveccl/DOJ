import axios from 'axios'
import wrap from './wrap'

export function getContests(params?: any) {
	return wrap(
		axios.get('/contest', { params })
	)
}

export function getContest(id: any) {
	return wrap(
		axios.get(`/contest/${id}`)
	)
}

export function postContest(data: any) {
	return wrap(
		axios.post('/contest', data)
	)
}

export function putContest(id: any, data: any) {
	return wrap(
		axios.put(`/contest/${id}`, data)
	)
}

export function delContest(id: any) {
	return wrap(
		axios.delete(`/contest/${id}`)
	)
}
