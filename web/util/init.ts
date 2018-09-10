import axios from 'axios'
import * as Cookie from 'js-cookie'

axios.defaults.baseURL = '/api'
axios.defaults.validateStatus = status => status < 500
axios.defaults.headers.common['token'] = Cookie.get('token')
