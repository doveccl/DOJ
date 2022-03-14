import axios from 'axios'
import moment from 'moment'
import Cookie from 'js-cookie'
import React, { createContext, useReducer } from 'react'
import { getConfig } from './model'
import { ILanguage, IUser } from './interface'

axios.defaults.baseURL = '/api'
axios.defaults.validateStatus = status => status <= 404
axios.defaults.headers.common.token = Cookie.get('token')

moment.defaultFormat = 'YYYY.MM.DD HH:mm:ss'

type Path = string | {
	url: string
	desc: string
}

interface Global {
	user?: IUser
	path?: Path[]
	languages?: ILanguage[]
}

const initial: Global = {}
const reducer = (o: Global, n: Global) => Object.assign({}, o, n)

export const GlobalContext = createContext(null as [Global, (_: Global) => void])
export const GlobalProvider: React.FC = ({ children }) => {
	const [state, dispatch] = useReducer(reducer, initial)
	return <GlobalContext.Provider value={[state, args => {
		if (args.path) updateTitle(args.path)
		// will update user but now there's no languages
		if (args.user && !global.languages)
			getConfig('languages').then(languages => dispatch({ languages }))
		dispatch(args) // might update user first and then update languages
	}]} children={children} />
}

function updateTitle(path: Path[]) {
	document.title = `${path.map(p => {
		return typeof p === 'string' ? p : p.desc
	}).join(' ')} - DOJ`
}
