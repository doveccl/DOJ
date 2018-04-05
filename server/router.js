const routes = [
	{ 'path': '/api/user/login', 'func': './api/user/login' }
]

for (let i of routes) {
	i.func = require(i.func)
}

module.exports = routes
