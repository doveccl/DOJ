import Login from './page/login.vue'
import Main from './page/main.vue'

import Home from './view/home.vue'

export default [
	{
		path: '/login',
		component: Login
	},
	{
		path: '/',
		component: Main,
		children: [
			{ path: 'home', component: Home },
			{ path: '*', redirect: 'home' }
		]
	}
]
