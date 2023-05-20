import { startServer } from './server'
import { startJudger } from './judger'

const s = process.argv.includes('--server')
const j = process.argv.includes('--judger')

s ? startServer().then(j ? startJudger : null) : j && startJudger()
