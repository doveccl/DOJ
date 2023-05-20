import axios from 'axios'
import wrap from './wrap'

export function getSubmissions(params?: any) {
  return wrap(
    axios.get('/submission', { params })
  )
}

export function getSubmission(id: any) {
  return wrap(
    axios.get(`/submission/${id}`)
  )
}

export function postSubmission(data: any) {
  return wrap(
    axios.post('/submission', data)
  )
}

export function putSubmission(id: any, data: any) {
  return wrap(
    axios.put(`/submission/${id}`, data)
  )
}

export function rejudgeSubmission(data: any) {
  return wrap(
    axios.put('/submission/rejudge', data)
  )
}
