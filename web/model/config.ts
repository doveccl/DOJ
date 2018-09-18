import axios from 'axios'
import wrap from './wrap'

export function getConfigs() {
	return wrap(
		axios.get('/config')
	)
}

export function putConfigs(data: any) {
	return wrap(
		axios.put('/config', data)
	)
}

export function getLanguages() {
	return wrap(
		axios.get('/config/languages')
	)
}
