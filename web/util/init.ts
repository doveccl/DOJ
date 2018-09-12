import axios from 'axios'
import * as Cookie from 'js-cookie'
import * as model from '../model'
import * as state from './state'

axios.defaults.baseURL = '/api'
axios.defaults.validateStatus = status => status < 500
axios.defaults.headers.common['token'] = Cookie.get('token')

state.addListener('update_languages', ({ user, languages }) => {
	if (user && languages.length === 0) {
		model.getLanguages()
			.then(languages => {
				state.updateState({ languages })
				state.removeListener('update_languages')
			})
			.catch(err => console.warn(err))
	}
})
