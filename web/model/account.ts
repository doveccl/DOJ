import axios from 'axios'
import Cookie from 'js-cookie'
import wrap from './wrap'

function setToken(token: string, expires: number) {
  Cookie.set('token', token, { expires })
  axios.defaults.headers.common.token = token
}

export function delToken() {
  Cookie.remove('token')
  delete axios.defaults.headers.common.token
}

export function getToken() {
  return Cookie.get('token')
}

export function hasToken() {
  return !!Cookie.get('token')
}

export function login(headers: any) {
  const { remember } = headers
  const exp = remember ? 7 : undefined
  return wrap(
    axios.get('/login', { headers }),
    ({ token }) => setToken(token, exp)
  )
}

export function register(data: any) {
  return wrap(
    axios.post('/register', data)
  )
}

export function getReset(user: string) {
  return wrap(
    axios.get('/reset', { params: { user } })
  )
}

export function putReset(data: any) {
  return wrap(
    axios.put(`/reset`, data)
  )
}
