const { get } = require('config')
if (get('server')) { require('./build') }
if (get('judger')) { require('./core') }
