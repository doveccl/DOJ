import { createTransport } from 'nodemailer'
import { config } from './config'

const	transporter = createTransport(config.mail)
const	from = `"DOJ System" <${config.mail.auth.user}>`

export function send(to: string, subject: string, text: string) {
	return transporter.sendMail({ from, to, subject, text })
}
