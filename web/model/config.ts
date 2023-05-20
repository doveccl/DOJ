import axios from 'axios'
import wrap from './wrap'

export function getConfig(id: any) {
  return wrap(
    axios.get(`/config/${id}`)
  )
}

export function putConfig(id: any, data: any) {
  return wrap(
    axios.put(`/config/${id}`, data)
  )
}
