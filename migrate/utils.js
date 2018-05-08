const readline = require('readline')
const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout
})
exports.input = (what, auto = '') => {
  let q = what
  if (auto != '') {
    q += ` (default is ${auto}): `
  } else { q += ': ' }
  return new Promise(res => {
    rl.question(q, a => res(a || auto))
  })
}
exports.end = () => rl.close()

const bcrypt = require('bcryptjs')
const dcrypto = require('dcrypto')
exports.pwd2to3 = (pwd2, s = 0, e = 'ab') => {
  let pwd = '123456'
  try {
    pwd = dcrypto.decrypt(pwd2, s, e)
  } catch (e) {
    console.warn('password error, use default password')
  }
  return bcrypt.hashSync(pwd)
}
