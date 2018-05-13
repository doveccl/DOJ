const mysql = require('mysql')
let mysql_connection = null
exports.mysql_connect = (h, u, p, d) => {
  return new Promise((res, rej) => {
    mysql_connection = mysql.createConnection({
      host: h, user: u, password: p, database: d
    })
    mysql_connection.connect(err => {
      if (err) { rej(err) } else { res() } 
    })
  })
}
exports.mysql_query = sql => {
  return new Promise((res, rej) => {
    mysql_connection.query(sql, (err, ans) => {
      if (err) { rej(err) } else { res(ans) }
    })
  })
}

const grid = require('gridfs-stream')
const mongoose = require('mongoose')
const schema = require('./schema')
let gridfs = null
exports.get_grid = () => gridfs
exports.model = name => mongoose.model(name)
exports.mongo_connect = async uri => {
  await mongoose.connect(uri, { reconnectTries: 0 })
  await mongoose.connection.db.dropDatabase()
  gridfs = new grid(mongoose.connection.db, mongoose.mongo)
  mongoose.model('User', schema.user)
  mongoose.model('Problem', schema.problem)
  mongoose.model('Contest', schema.contest)
  mongoose.model('Submission', schema.submission)
  mongoose.model('Language', schema.language)
}
exports.put_zip = async (zip, name) => {
  return new Promise((res,rej) => {
    let stream = gridfs.createWriteStream({
      filename: name,
      content_type: 'application/zip'
    })
    stream.on('close', res)
    stream.on('error', rej)
    zip.generateNodeStream({
      type: 'nodebuffer', streamFiles: true
    }).pipe(stream)
  })
}

exports.close = () => {
  mongoose.disconnect()
  if (mysql_connection) {
    mysql_connection.end()
  }
}
