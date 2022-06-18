import App from '@/App.vue'
import { router } from '@/route'
import { i18n } from '@stores/i18n'

createApp(App)
  .use(i18n)
  .use(router)
  .use(createPinia())
  .mount('#app')
