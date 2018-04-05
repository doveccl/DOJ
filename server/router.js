const routes = [
	{ 'path': '/api/user/login', 'func': './api/user/login' },
	{ 'path': '/api/user/info', 'func': './api/user/info' },
]

for (let i of routes) {
	i.func = require(i.func)
}

module.exports = routes
