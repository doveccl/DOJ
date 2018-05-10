const db = require('./db')
const utils = require('./utils')
const { readFileSync, writeFileSync } = require('fs')
const confile = require('path').join(__dirname, 'config.tmp')

const main = async () => {
  let config = {}
  try {
    config = JSON.parse(readFileSync(confile))
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
      'Data directory', config.dd || ''
    )
    config.seed = parseInt(await utils.input(
      'Password seed', config.seed || '0'
    ))
    config.ab = await utils.input(
      'Password salt', config.ab || 'ab'
    )
    console.log('Saving config to tmp file ...')
    writeFileSync(confile, JSON.stringify(config, null, 2))
  }
  console.log('Config:', config)

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
  console.log('Id to ObjectId', uidx)

}

const end = err => {
  utils.end()
  db.close()
  if (err) { console.error(err) }
}
main().then(end).catch(e => end(e))
