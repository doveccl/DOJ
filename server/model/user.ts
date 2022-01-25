import { hashSync } from 'bcryptjs'
import { model, Document, Schema } from 'mongoose'

import { Group, IUser } from '../../common/interface'

export type DUser = IUser<Schema.Types.ObjectId, Date> & Document

const schema = new Schema({
	name: {
		type: String,
		unique: true,
		required: true,
		match: /^[a-zA-Z0-9][a-zA-Z0-9_]{2,14}$/
	},
	mail: {
		type: String,
		unique: true,
		required: true,
		match: /^[\w.]+@(?:[a-z0-9]+(?:-[a-z0-9]+)*\.)+[a-z]{2,}$/
	},
	group: {
		type: Number,
		min: 0, max: 2,
		default: 0
	},
	password: {
		type: String,
		required: true
	},
	solve: {
		type: Number,
		min: 0, default: 0
	},
	submit: {
		type: Number,
		min: 0, default: 0
	},
	introduction: {
		type: String,
		maxlength: 200,
		default: ''
	}
}, {
	versionKey: false,
	timestamps: true,
	collation: {
		locale: 'en_US',
		strength: 2
	}
})

schema.index({ solve: -1, submit: 1 })

export const User = model<DUser>('user', schema)
