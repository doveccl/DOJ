import axios from 'axios'
import wrap from './wrap'

export function getConfigs() {
	return wrap(
		axios.get('/config')
	)
}

export function getLanguages() {
	return wrap(
		axios.get('/config/languages')
	)
}
