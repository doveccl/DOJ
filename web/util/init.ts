import axios from 'axios'

axios.defaults.baseURL = '/api'
axios.defaults.validateStatus = status => status < 500
