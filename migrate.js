const readline = require('readline')
const dcrypto = require('dcrypto')
const bcrypt = require('bcrypt')
const mysql = require('mysql')

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout
})

const input = (ques) => {
  return new Promise((res) => {
    rl.question(ques, ans => res(ans))
  })
}

let mysql_connection;
const mysql_connect = (h, u, p, d) => {
  return new Promise((res, rej) => {
    mysql_connection = mysql.createConnection({
      host: h, user: u, password: p, database: d
    })
    mysql_connection.connect(err => {
      if (err) { rej(err) } else { res() } 
    })
  })
}
const mysql_query = sql => {
  return new Promise((res, rej) => {
    mysql_connection.query(sql, (err, ans) => {
      if (err) { rej(err) } else { res(ans) }
    })
  })
}

const main = async () => {
  let mysql_host = await input('MySQL Host: ') || 'localhost'
  let mysql_usr = await input('MySQL Username: ') || 'root'
  let mysql_pwd = await input('MySQL Password: ')
  let mysql_db = await input('MySQL Database: ') || 'doj'
  try {
    await mysql_connect(mysql_host, mysql_usr, mysql_pwd, mysql_db)
  } catch (e) {
    console.error('connect error')
    process.exit(1)
  }

  console.log(await mysql_query('select 1+1 as ans'))
}

main().then(() => {
  rl.close()
  mysql_connection.end()
})
