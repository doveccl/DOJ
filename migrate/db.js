const mysql = require('mysql')
exports.mysql_connection = null
exports.mysql_connect = (h, u, p, d) => {
  return new Promise((res, rej) => {
    exports.mysql_connection = mysql.createConnection({
      host: h, user: u, password: p, database: d
    })
    exports.mysql_connection.connect(err => {
      if (err) { rej(err) } else { res() } 
    })
  })
}
exports.mysql_query = sql => {
  return new Promise((res, rej) => {
    exports.mysql_connection.query(sql, (err, ans) => {
      if (err) { rej(err) } else { res(ans) }
    })
  })
}

const mongoose = require('mongoose')
const schema = require('./schema')
exports.mongo_connection = null
exports.model = {
  user: null
}
exports.mongo_connect = uri => {
  exports.mongo_connection = mongoose.createConnection(uri)
  exports.model.user = exports.mongo_connection.model('User', schema.user)
}
