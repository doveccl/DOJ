export const useUserStore = defineStore('user', () => {
  const token = useLocalStorage('token', '')
  const info = reactive({ ID: 0, Name: '', Mail: '', Group: 0, Status: '' })

  watchEffect(() => {
    const { common } = axios.defaults.headers
    token.value ? (common.token = token.value) : delete common.token
    token.value && axios.get('/self').then(r => Object.assign(info, r.data), console.warn)
  })

  async function login({ user = '', pass = '' }) {
    const r = await axios.post('/login', { user, pass })
    return (token.value = r.data.token)
  }

  function logout() {
    token.value = ''
    info.ID = info.Group = 0
  }

  return { info, token, login, logout }
})
