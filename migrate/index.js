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
    console.log('Saving config:', config)
    writeFileSync(confile, JSON.stringify(config))
  }

  try {
    db.mysql_connect(config.mh, config.mu, config.mp, config.md)
    db.mongo_connect(config.uri)
  } catch (e) {
    return console.error(e)
  }

  // start migrate
}

main().then(() => utils.end()).catch(e => console.error(e))
