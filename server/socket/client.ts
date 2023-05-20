import WebSocket from 'ws'
import { Group, Status } from '../../common/interface'
import { Pack } from '../../common/pack'
import { diffGroup } from '../../common/user'
import { contest, user } from '../middleware/fetch'
import { Submission } from '../model/submission'
import { verify } from '../util/jwt'
import { logSocket } from '../util/log'

const wmap = new Map<string, WebSocket[]>()

export async function update(pack: Pack) {
  wmap.get(pack._id)?.forEach(ws => {
    ws.send(JSON.stringify(pack))
  })
  if (!pack.pending) {
    wmap.delete(pack._id)
    await Submission.findByIdAndUpdate(pack._id, pack)
  }
}

export function routeClient(ws: WebSocket) {
  ws.on('message', async raw => {
    const data = JSON.parse(raw.toString())
    if (data.id && data.token) {
      const s = await Submission.findById(data.id)
      if (!s) return
      const c = s.cid ? await contest(s.cid) : null
      const u = await user(verify(data.token).id)
      if (!u) return
      logSocket.info('User query:', u.name, data.id)
      if (s.result.status !== Status.WAIT) {
        ws.send(JSON.stringify(s))
      } else if (diffGroup(u, Group.admin) || c?.freezeAt && new Date() < c.freezeAt) {
        if (wmap.has(data.id))
          wmap.get(data.id)?.push(ws)
        else
          wmap.set(data.id, [ws])
      } else {
        ws.close()
      }
    }
  })
}
