import axios from 'axios'
import * as Cookie from 'js-cookie'

import { getConfig } from '../model'
import { addListener, removeListener, updateState } from './state'

axios.defaults.baseURL = '/api'
axios.defaults.validateStatus = (status) => status <= 404
axios.defaults.headers.common.token = Cookie.get('token')

addListener('update_languages', ({ user, languages }) => {
	if (user && languages.length === 0) {
		getConfig('languages')
			.then((list) => {
				updateState({ languages: list })
				removeListener('update_languages')
			})
	}
})
