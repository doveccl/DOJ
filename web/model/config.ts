import axios from 'axios'
import wrap from './wrap'

export function getConfigs() {
	return wrap(
		axios.get('/config')
	)
}
