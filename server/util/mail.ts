import { get } from 'config'
import { createTransport } from 'nodemailer'

const transporter = createTransport(get('mail'))

export function send(to: string, subject: string, text: string) {
	return transporter.sendMail({ from: 'DOJ', to, subject, text })
}
