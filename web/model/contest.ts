import axios from 'axios'
import wrap from './wrap'

export function getContests(params?: any) {
	return wrap(
		axios.get('/contest', { params })
	)
}

export function getContest(id: string) {
	return wrap(
		axios.get(`/contest/${id}`)
	)
}
