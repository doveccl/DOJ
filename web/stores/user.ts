import { useLocalStorage } from '@vueuse/core'

export const useUserStore = defineStore('user', () => {
  const token = useLocalStorage('token', '')
  const info = reactive({
    ID: 0,
    Name: '',
    Mail: '',
    Group: 0,
    Status: ''
  })

  watchEffect(() => {
    token.value ?
      axios.defaults.headers.common.token = token.value :
      delete axios.defaults.headers.common.token
    token.value && axios.get('/api/self')
      .then(r => Object.assign(info, r.data))
      .catch(e => console.warn('get self info', e))
  })

  async function login(user: string, pass: string) {
    const r = await axios.post('/api/login', { user, pass })
    return token.value = r.data.token
  }

  return { info, token, login }
})
