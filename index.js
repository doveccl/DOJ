const config = require('./config')
const server = require('./server')

server.start(config.server)
