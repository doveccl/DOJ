import { hashSync } from 'bcryptjs'
import { decrypt } from 'dcrypto'
import { createInterface } from 'readline'

const rl = createInterface(process.stdin, process.stdout)

export const close = () => rl.close()
export const input = (ques: string, defa?: string) => new Promise<string>(
	(r) => rl.question(ques + (defa ? ` (${defa}): ` : ': '), (a) => r(a || defa))
)

let s = 0
let e = 'ab'
export const setSE = (S: number, E: string) => (s = S, e = E)
export const pwd2to3 = (pwd2: string) => {
	let pwd = '123456'
	try {
		pwd = decrypt(pwd2, s, e)
	} catch (err) {
		console.warn(`password decrypt error, use '${pwd}'`)
	}
	return hashSync(pwd)
}
