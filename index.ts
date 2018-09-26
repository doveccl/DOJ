import judger from './judger'
import server from './server'

if (process.argv.includes('--server')) { server() }
if (process.argv.includes('--judger')) { judger() }
