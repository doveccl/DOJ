import WebSocket from 'ws'
import config from '../../config'
import { update } from './client'
import { logSocket } from '../util/log'
import { SE } from '../../common/pack'
import { Status } from '../../common/interface'
import { problem } from '../middleware/fetch'
import { DSubmission, Submission } from '../model/submission'

interface ParsedSubmission {
  _id: any
  language: number,
  code: string
  timeLimit: number
  memoryLimit: number
  data: any
}

let judgers: WebSocket[] = []
let submissions: ParsedSubmission[] = []

function dispatchSubmission() {
  while (judgers.length && submissions.length) {
    const judger = judgers.shift()
    const submission = submissions.shift()
    judger?.send(JSON.stringify(submission))
  }
}

function addJudger(ws: WebSocket, count = 1) {
  while (count--) judgers.push(ws)
  dispatchSubmission()
}

export function routeJudger(ws: WebSocket) {
  let name = ''
  ws.on('message', raw => {
    const data = JSON.parse(raw.toString())
    if (data.name && data.secret && data.concurrent) {
      if (config.secret === data.secret) {
        logSocket.info('Add judger:', name = data.name)
        addJudger(ws, data.concurrent)
      } else {
        logSocket.warn('Invalid judger:', data.name)
        ws.close(2000, 'invalid judger')
      }
    } else if (!name) {
      ws.close(2000, 'invalid judger')
    } else if (data._id) {
      data.pending || addJudger(ws)
      update(data)
    }
  }).on('close', () => {
    logSocket.info('Del judger:', name)
    judgers = judgers.filter(w => w !== ws)
  })
  // fetch all pending submissions to judge
  Submission.find({ 'result.status': Status.WAIT })
    .then(list => list.forEach(doJudge))
}

async function parseSubmission(s: DSubmission) {
  const p = await problem(s.pid)
  if (!p) throw new Error('problem not found')
  const { _id, language, code } = s
  const { timeLimit, memoryLimit, data } = p
  return { _id, language, code, timeLimit, memoryLimit, data }
}

export function doJudge(s: DSubmission) {
  parseSubmission(s).then(p => {
    submissions.push(p)
    dispatchSubmission()
  }).catch(e => {
    update(SE(s._id, e.message))
  })
}
