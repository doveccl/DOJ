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

const mongoose = require('mongoose')
const schema = require('./schema')
exports.model = name => mongoose.model(name)
exports.mongo_connect = async uri => {
  await mongoose.connect(uri, { reconnectTries: 0 })
  await mongoose.connection.db.dropDatabase()
  mongoose.model('User', schema.user)
}

exports.close = () => {
  mongoose.disconnect()
  mysql_connection.end()
}
