import jwt from 'jsonwebtoken'
import { config } from './config'

export function sign(
	payload: string | object | Buffer,
	secretOrPrivateKey: string | Buffer = config.secret,
	options: jwt.SignOptions = { expiresIn: '7d' }
) {
	return new Promise<string>((resolve, reject) => {
		jwt.sign(payload, secretOrPrivateKey, options, (error, data) => {
			if (error) {
				reject(error)
			} else {
				resolve(data)
			}
		})
	})
}

export function verify(
	token: string,
	secretOrPrivateKey: string | Buffer = config.secret
) {
	return new Promise<string | object>((resolve, reject) => {
		jwt.verify(token, secretOrPrivateKey, (error, data) => {
			if (error) {
				reject(error)
			} else {
				resolve(data)
			}
		})
	})
}
