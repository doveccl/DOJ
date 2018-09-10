import axios from 'axios'
import wrap from './wrap'

export function getSelfInfo() {
	return wrap(
		axios.get('/user/info')
	)
}
