import { model, Document, Schema } from 'mongoose'
import { IProblem } from '../../common/interface'

export type DProblem = IProblem<Schema.Types.ObjectId, Date> & Document

const belong = new Schema({
	id: Schema.Types.ObjectId,
	key: String
}, {
	_id: false
})

const schema = new Schema({
	title: {
		type: String,
		required: true,
		maxlength: 30
	},
	content: {
		type: String,
		required: true
	},
	tags: {
		type: [String],
		default: [] as string[]
	},
	timeLimit: {
		type: Number,
		min: 0, default: 1000
	},
	memoryLimit: {
		type: Number,
		min: 0, default: 64000
	},
	solve: {
		type: Number,
		min: 0, default: 0
	},
	submit: {
		type: Number,
		min: 0, default: 0
	},
	data: {
		type: Schema.Types.ObjectId
	},
	contest: {
		type: belong
	}
}, {
	versionKey: false,
	timestamps: true,
	collation: {
		locale: 'en_US',
		strength: 2
	}
})

schema.index(
	{ 'contest.id': 1, 'contest.key': 1 },
	{ unique: true, sparse: true }
)

export const Problem = model<DProblem>('problem', schema)
