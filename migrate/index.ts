/**
 * migrate DOJ v2 to v3
 * optional dependencies are required:
 *   yarn add -O dcrypto mysql @types/mysql
 * to start migration in project root, simply run command:
 *   npx ts-node migrate # OR yarn ts-node migrate
 */

import JSZip from 'jszip'

import { closeSync, openSync, outputFile, pathExists, readFileSync, readSync } from 'fs-extra'
import { connect } from 'mongoose'
import { createConnection, Connection } from 'mysql'
import { join } from 'path'

import { ContestType } from '../common/interface'
import { Contest } from '../server/model/contest'
import { File } from '../server/model/file'
import { Problem } from '../server/model/problem'
import { Submission } from '../server/model/submission'
import { User } from '../server/model/user'
import { input, pwd2to3, setSE } from './function'

let data: string
let upload: string
let mysql: Connection

const mysqlQuery = (q: string) => new Promise<any>((res, rej) => {
  mysql.query(q, (error, results) => {
    if (error) { rej(error) } else { res(results) }
  })
})

async function parseMD(s: string, imgname?: string) {
  const matchs = s.match(/!\[[^\]]+\]\(\?img=\w+\)/g)
  if (!matchs) { return s }
  let res = s
  let type = 'image'
  for (const match of matchs) {
    const md5 = match.replace(/(^[^=]+=|\)$)/g, '')
    const buf = Buffer.from([0, 0])
    try {
      const imgfd = openSync(join(upload, md5), 'r')
      readSync(imgfd, buf, 0, 2, 0)
      closeSync(imgfd)
      if (buf[0] === 0xff && buf[1] === 0xd8) {
        type = 'image/jpeg'
      } else if (buf[0] === 0x89 && buf[1] === 0x50) {
        type = 'image/png'
      } else if (buf[0] === 0x47 && buf[1] === 0x49) {
        type = 'image/gif'
      } else if (buf[0] === 0x42 && buf[1] === 0x4d) {
        type = 'image/bmp'
      }
      const fid = await File.create(
        join(upload, md5),
        imgname || 'image'
      )
      res = res.replace(match, `[[ IMG id="${fid}" ]]`)
    } catch (e) {
      console.error('problem image add error')
      res = res.replace(match, '`error image`')
    }
  }
  return res
}

async function addData(pid: string) {
  console.log('add data:', pid)
  if (!await pathExists(join(data, pid))) {
    console.warn('miss data:', pid)
    return undefined
  } else {
    let i = 0
    const zip = new JSZip()
    do {
      try {
        zip.file(`${i}.in`, readFileSync(join(data, pid, `${i}.in`)))
        zip.file(`${i}.out`, readFileSync(join(data, pid, `${i}.out`)))
      } catch (e) {
        console.warn('wrong data:', pid)
        return undefined
      }
      i++
    } while (
      (await pathExists(join(data, pid, `${i}.in`))) &&
      (await pathExists(join(data, pid, `${i}.out`)))
    )
    await outputFile(
      join(data, pid, `P${pid}-data.zip`),
      await zip.generateAsync({ type: 'nodebuffer' })
    )
    return await File.create(
      join(data, pid, `P${pid}-data.zip`),
      `P${pid}-data.zip`
    )
  }
}

async function main() {
  const uid: { [index: string]: string } = {}
  const pid: { [index: string]: string } = {}
  const cid: { [index: string]: string } = { 0: null }
  console.log('--- user migration ---')
  const users = await mysqlQuery('select * from users')
  await User.deleteMany({})
  for (const user of users) {
    console.log('add user:', user.name)
    const u = await User.create({
      name: user.name,
      mail: user.mail,
      password: pwd2to3(user.password),
      introduction: user.sign,
      group: user.admin,
      createdAt: user.reg_time,
      updatedAt: new Date()
    })
    uid[user.id] = u._id
  }
  console.log('--- contest migration ---')
  const contests = await mysqlQuery('select * from contests')
  await Contest.deleteMany({})
  for (const contest of contests) {
    console.log('add contest:', contest.id, contest.name)
    const c = await Contest.create({
      title: `${contest.id} - ${contest.name}`,
      type: ContestType.OI,
      description: `Contest ${contest.id} - ${contest.name}`,
      startAt: contest.start_time,
      endAt: new Date(60 * 1000 * contest.time + contest.start_time.getTime())
    })
    cid[contest.id] = c._id
  }
  console.log('--- problem migration ---')
  const problems = await mysqlQuery('select * from problems')
  await Problem.deleteMany({})
  for (const problem of problems) {
    console.log('add problem:', problem.id, problem.name)
    if (problem.hide) {
      console.log('ignore hide problem')
      continue
    }
    let content = ''
    const imgname = `P${problem.id}-IMG`
    problem.id = String(problem.id)
    if (isNaN(problem.create_time)) {
      problem.create_time = 0
    }
    if (problem.description) {
      content += '## Description\n'
      content += await parseMD(problem.description, imgname)
      content += '\n'
    }
    if (problem.input) {
      content += '## Input Format\n'
      content += await parseMD(problem.input, imgname)
      content += '\n'
    }
    if (problem.output) {
      content += '## Output Format\n'
      content += await parseMD(problem.output, imgname)
      content += '\n'
    }
    if (await pathExists(join(data, problem.id, 's.in'))) {
      content += '## Sample Input\n```\n'
      content += readFileSync(join(data, problem.id, 's.in'))
      content += '\n```\n'
    }
    if (await pathExists(join(data, problem.id, 's.out'))) {
      content += '## Sample Output\n```\n'
      content += readFileSync(join(data, problem.id, 's.out'))
      content += '\n```\n'
    }
    if (problem.hint) {
      content += '## Hint\n'
      content += await parseMD(problem.hint, imgname)
      content += '\n'
    }
    const p = await Problem.create({
      title: `P${problem.id} - ${problem.name}`,
      content: content || 'no content for this problem',
      tags: problem.tags ? problem.tags.split('|') : [],
      timeLimit: problem.time / 1000,
      memoryLimit: 1024 * 1024 * problem.memory / 1000,
      createdAt: problem.create_time,
      updatedAt: problem.create_time,
      data: await addData(problem.id),
      contest: problem.cid ? {
        id: cid[problem.cid],
        key: problem.id
      } : null
    })
    pid[problem.id] = p._id
  }
  console.log('--- submission migration ---')
  const submits = await mysqlQuery('select * from submit')
  const lan = [
    0, // C -> C
    1, // C++ -> C++
    -1, // Pascal -> drop
    2, // Python -> Pyhon 2.x
    1 // C++11 -> C++
  ]
  await Submission.deleteMany({})
  for (const submit of submits) {
    console.log('add submit:', submit.id)
    console.log('uid:', submit.uid)
    console.log('pid:', submit.pid)
    if (lan[submit.language] < 0) {
      console.warn('drop unsupported languages')
      continue
    }
    if (!uid[submit.uid]) {
      console.warn('no such user')
      continue
    }
    if (!pid[submit.pid]) {
      console.warn('no such problem')
      continue
    }
    if (submit.cid && !cid[submit.cid]) {
      console.warn('no such contest')
      continue
    }
    await Submission.create({
      uid: uid[submit.uid],
      pid: pid[submit.pid],
      cid: cid[submit.cid],
      language: lan[submit.language],
      code: submit.code,
      createdAt: submit.submit_time
    })
  }
  process.exit(0)
}

(async () => {
  const myuri = await input('mysql uri', 'mysql://root@127.0.0.1/doj')
  const mouri = await input('mongo uri', 'mongodb://127.0.0.1:27017/doj')
  const seed = Number(await input('v2 password seed', '0'))
  const enc = await input('v2 password enc', 'ab')
  data = await input('data path', '/home/doj')
  upload = await input('img upload path', '/home/www/doj/upload')
  mysql = createConnection(myuri)
  mysql.connect((mye) => {
    if (mye) {
      console.error(mye)
      process.exit(-1)
    }
    connect(mouri, (moe) => {
      if (moe) {
        console.error(moe)
        process.exit(-1)
      }
      setSE(seed, enc)
      main()
    })
  })
})()
