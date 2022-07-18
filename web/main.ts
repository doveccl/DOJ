import App from '@/App.vue'
import { router } from '@/route'
import { i18n } from '@stores/i18n'
import { AxiosError } from 'axios'

axios.defaults.baseURL = '/api'
axios.interceptors.response.use(undefined, e => {
  if (e instanceof AxiosError) {
    return Promise.reject(e.response?.data?.message ?? e.message)
  } else if (e instanceof Error) {
    return Promise.reject(e.message)
  } else {
    return Promise.reject(`${e}`)
  }
})

createApp(App)
  .use(i18n)
  .use(router)
  .use(createPinia())
  .mount('#app')
