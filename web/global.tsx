import axios from 'axios'
import moment from 'moment'
import Cookie from 'js-cookie'
import { createContext, useReducer } from 'react'
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

declare module globalThis {
  var user: IUser
  var path: Path[]
  var languages: ILanguage[]
}

type G = Partial<typeof globalThis>

const initial: G = {}
const reducer = (o: G, n: G) => Object.assign({}, o, n)

export const GlobalContext = createContext<[G, (_: G) => void]>([{}, () => {}])
export function GlobalProvider({ children = <></> }) {
  const [state, dispatch] = useReducer(reducer, initial)
  return <GlobalContext.Provider value={[state, args => {
    if (args.path) updateTitle(args.path)
    // will update user but now there's no languages
    if (args.user && !globalThis.languages)
      getConfig('languages').then(languages => dispatch({ languages }))
    dispatch(args) // might update user first and then update languages
  }]} children={children} />
}

function updateTitle(path: Path[]) {
  document.title = `${path.map(p => {
    return typeof p === 'string' ? p : p.desc
  }).join(' ')} - DOJ`
}
