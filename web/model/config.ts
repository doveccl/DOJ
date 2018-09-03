import axios from 'axios'
import wrap from './wrap'

export function getConfig() {
	return wrap(
		axios.get('/config')
	)
}
