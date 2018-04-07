import Vue from 'vue'
import VueRouter from 'vue-router'
import iView from 'iview'
import 'iview/dist/styles/iview.css'

Vue.use(VueRouter)
Vue.use(iView)

import app from './app.vue'
import routes from './router'
const router = new VueRouter({
	mode: 'history',
	routes: routes
})

router.beforeEach((to, from, next) => {
	iView.LoadingBar.start()
	next()
})

router.afterEach(() => {
	iView.LoadingBar.finish()
})

new Vue({
	el: '#app', router,
	render: h => h(app)
})
