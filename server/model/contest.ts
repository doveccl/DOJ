import { Schema, Document, model } from 'mongoose'

export enum ContestType { OI, ICPC }

export interface IContest extends Document {
	title: string
	description: string
	type: ContestType
	startAt: Date
	endAt: Date
	freezeAt?: Date
}

const schema = new Schema({
	title: {
		type: String,
		unique: true,
		required: true,
		maxlength: 30
	},
	description: {
		type: String,
		default: ''
	},
	type: {
		type: Number,
		required: true,
		min: 0, max: 1
	},
	startAt: {
		type: Date,
		required: true
	},
	endAt: {
		type: Date,
		required: true
	},
	freezeAt: {
		type: Date,
		default: new Date(0)
	}
}, {
	versionKey: false,
	timestamps: true
})

export default model<IContest>('contest', schema)
