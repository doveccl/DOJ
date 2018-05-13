const db = require('./db')
const utils = require('./utils')

const jszip = require('jszip')
const yaml = require('js-yaml')

const { join } = require('path')
const { readFileSync, writeFileSync } = require('fs')

const confile = join(__dirname, 'config.tmp')

const main = async () => {
  let config = {}
  try {
    config = yaml.load(readFileSync(confile))
    if (await utils.input(
      'Load config from saved tmp file? (Y/n)'
    ) === 'n') {
      throw new Error('need to re-question config')
    }
  } catch (e) {
    config.mh = await utils.input(
      'MySQL host', config.mh || 'localhost'
    )
    config.mu = await utils.input(
      'MySQL user', config.mu || 'root'
    )
    config.mp = await utils.input(
      'MySQL password', config.mp || ''
    )
    config.md = await utils.input(
      'MySQL database', config.md || 'doj'
    )
    config.uri = await utils.input(
      'Mongo URI', config.uri || 'mongodb://localhost/doj'
    )
    config.dd = await utils.input(
      'Data directory', config.dd || '/home/doj/data'
    )
    config.seed = parseInt(await utils.input(
      'Password seed', config.seed || '0'
    ))
    config.ab = await utils.input(
      'Password salt', config.ab || 'ab'
    )
    console.log('Saving config to tmp file ...')
    writeFileSync(confile, yaml.dump(config))
  }

  console.log('Connecting to MySQL ...')
  await db.mysql_connect(config.mh, config.mu, config.mp, config.md)
  console.log('Connecting to MongoDB ...')
  await db.mongo_connect(config.uri)

  let users = await db.mysql_query('SELECT * FROM users')
  let modelUser = db.model('User')
  let uidx = {}
  for (let user of users) {
    console.log('Add user:', user.name)
    uidx[user.id] = (await modelUser.create({
      name: user.name,
      mail: user.mail,
      admin: user.admin,
      password: utils.pwd2to3(
        user.password,
        config.seed, config.ab
      ),
      introduction: user.sign,
      join_time: user.reg_time
    }))._id
  }

  let contest_problems = {}

  let problems = await db.mysql_query('SELECT * FROM problems')
  let modelProblem = db.model('Problem')
  let pidx = {}
  for (let problem of problems) {
    console.log('Add problem:', problem.id)
    let pd = join(config.dd, `${problem.id}`)
    let zip = new jszip()
    let con = {
      time: problem.time,
      memory: problem.memory,
      interactor: null,
      checker: null 
    }
    let i = 0
    let flag = true
    let data = zip.folder('data')
    while (flag) {
      try {
        let inf = readFileSync(join(pd, `${i}.in`))
        let ouf = readFileSync(join(pd, `${i}.out`))
        data.file(`${i}.in`, inf.toString().replace(/\r\n/g, '\n'))
        data.file(`${i}.out`, ouf.toString().replace(/\r\n/g, '\n'))
        i++
      } catch(e) {
        flag = false
      }
    }
    con.count = i
    zip.file('config.yml', yaml.dump(con))
    let text = '#Description\n\n' + problem.description + '\n\n'
    if (problem.input) {
      text += `#Input Format\n\n${problem.input}\n\n`
    }
    if (problem.output) {
      text += `#Output Format\n\n${problem.output}\n\n`
    }
    try {
      text += `#Sample Input\n\n
\`\`\`
${readFileSync(join(pd, 's.in'))}
\`\`\`\n\n`
      text += `#Sample Output\n\n
\`\`\`
${readFileSync(join(pd, 's.out'))}
\`\`\`\n\n`
    } catch(e) { console.error('cant find sample file') }
    if (problem.hint) {
      text += `#Hint\n\n${problem.hint}\n\n`
    }
    pidx[problem.id] = (await modelProblem.create({
      title: problem.name,
      content: text,
      enable: problem.hide ? false : true
    }))._id
    if (problems.cid != 0) {
      if (contest_problems[problem.cid] === undefined) {
        contest_problems[problem.cid] = []
      }
      contest_problems[problem.cid].push(pidx[problem.id])
    }
    let d = await db.put_zip(zip, `${pidx[problem.id]}`)
    await modelProblem.findById(pidx[problem.id]).update({
      data: d._id
    }).exec()
  }

  let contests = await db.mysql_query('SELECT * FROM contests')
  let modelContest = db.model('Contest')
  let cidx = {}
  for (let contest of contests) {
    console.log('Add contest:', contest.id)
    cidx[contest.id] = (await modelContest.create({
      title: contest.name,
      instruction: '',
      problems: contest_problems[contest.id],
      start_time: contest.start_time,
      end_time: new Date(
        contest.start_time.getTime() +
        1000 * 60 * contest.time
      ),
      type: 'oi'
    }))._id
  }

  let modelLanguage = db.model('Language')
  let lidx =  []
  lidx.push((await modelLanguage.create({
    name: 'C',
    config: {}
  }))._id)
  lidx.push((await modelLanguage.create({
    name: 'C++',
    config: {}
  }))._id)
  lidx.push((await modelLanguage.create({
    name: 'Pascal',
    config: {}
  }))._id)
  lidx.push((await modelLanguage.create({
    name: 'Python2',
    config: {}
  }))._id)

  const status = [
    'wait',
    'ac', 'wa', 'tle', 'mle', 're', 'ce',
    'se', 'se', 'se', 'se', 'se', 'se'
  ]

  let submissions = await db.mysql_query('SELECT * FROM submit')
  let modelSubmission = db.model('Submission')
  for (let submission of submissions) {
    console.log('Add record:', submission.id)
    let res = {
      status: status[submission.res],
      time: 0, memory: 0
    }, cases = []
    try {
      let arr = JSON.parse(submission.result)
      for (let i of arr) {
        res.time = Math.max(res.time, i.time)
        res.memory = Math.max(res.memory, i.memory)
        cases.push({
          status: status[i.res],
          time: i.time, memory: i.memory
        })
      }
    } catch(e) {
      console.log('Analyze result error')
    }
    await modelSubmission.create({
      user_id: uidx[submission.uid],
      problem_id: pidx[submission.pid],
      contest_id: submission.cid ? cidx[submission.cid] : null,
      time: submission.submit_time,
      language: lidx[
        submission.language == 4 ? 1 : submission.language
      ],
      code: submission.code,
      result: res,
      cases: cases
    })
  }
}

const end = err => {
  utils.end()
  db.close()
  if (err) { console.error(err) }
}
main().then(end).catch(e => end(e))
