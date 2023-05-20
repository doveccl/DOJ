import axios from 'axios'
import wrap from './wrap'

export function getSelfInfo() {
  return wrap(
    axios.get('/user/info')
  )
}

export function getUsers(params?: any) {
  return wrap(
    axios.get('/user', { params })
  )
}

export function postUser(data: any) {
  return wrap(
    axios.post('/user', data)
  )
}

export function inviteUser(data: any) {
  return wrap(
    axios.post('/user/invite', data)
  )
}

export function putUser(id: any, data: any) {
  return wrap(
    axios.put(`/user/${id}`, data)
  )
}

export function delUser(id: any) {
  return wrap(
    axios.delete(`/user/${id}`)
  )
}
