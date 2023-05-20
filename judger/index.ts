import WebSocket from 'ws'
import { initPath } from './path'
import config from '../config'
import { judge } from './judge'
import { logJudger } from '../server/util/log'
import { SE } from '../common/pack'

export function startJudger() {
  initPath()

  const { host, name, secret, concurrent } = config
  const ws = new WebSocket(`ws://${host}/ws?judger`)
  logJudger.info('connect to server:', ws.url)

  ws.onerror = e => logJudger.warn('connect error:', e.message)
  ws.onclose = () => setTimeout(startJudger, 5000) // reconnect
  ws.onopen = () => {
    logJudger.info('connected, register as judger')
    ws.send(JSON.stringify({ name, secret, concurrent }))
    ws.on('message', raw => {
      const data = JSON.parse(raw.toString())
      judge(data, ws).then(res => {
        ws.send(JSON.stringify(res))
      }).catch(e => {
        logJudger.warn('judge error:', e?.message ?? e)
        e?.stack && logJudger.debug(e.stack)
        ws.send(JSON.stringify(SE(data._id, e)))
      })
    })
  }
}
