import axios from 'axios'
import wrap from './wrap'

export function getFiles(params?: any) {
	return wrap(
		axios.get('/file', { params })
	)
}

export function putFile(id: any, data?: any) {
	return wrap(
		axios.put(`/file/${id}`, data)
	)
}

export function delFile(id: any) {
	return wrap(
		axios.delete(`/file/${id}`)
	)
}
