import axios from 'axios'
import wrap from './wrap'

export function getProblems(params?: any) {
	return wrap(
		axios.get('/problem', { params })
	)
}
