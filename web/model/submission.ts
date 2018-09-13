import axios from 'axios'
import wrap from './wrap'

export function getSubmissions(params?: any) {
	return wrap(
		axios.get('/submission', { params })
	)
}

export function postSubmission(data: any) {
	return wrap(
		axios.post('/submission', data)
	)
}
