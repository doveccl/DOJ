import { model, Document, Schema } from 'mongoose'
import { IContest } from '../../common/interface'

export type DContest = IContest<Schema.Types.ObjectId, Date> & Document

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

export const Contest = model<DContest>('contest', schema)
