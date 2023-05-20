import jwt from 'jsonwebtoken'
import config from '../../config'

export function sign(data: object, expiresIn = '7d') {
  return jwt.sign(data, config.secret, { expiresIn })
}

export function verify(token: string) {
  return jwt.verify(token, config.secret) as jwt.JwtPayload
}
