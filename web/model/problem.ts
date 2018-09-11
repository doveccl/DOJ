import axios from 'axios'
import wrap from './wrap'

export function getProblems(params?: any) {
	return wrap(
		axios.get('/problem', { params })
	)
}

export function getProblem(id: string) {
	return wrap(
		axios.get(`/problem/${id}`)
	)
}
