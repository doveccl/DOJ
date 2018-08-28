import { Schema, Document, model } from 'mongoose'

export interface IProblem extends Document {
	title: string
	content: string
	tags: string[]
	timeLimit: number
	memoryLimit: number
	solve: number
	submit: number
	data?: Schema.Types.ObjectId
	contest?: {
		id: Schema.Types.ObjectId
		key: string
	}
}

const belong = new Schema({
	id: Schema.Types.ObjectId,
	key: String
}, { _id: false })

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
		default: <string[]>[]
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
		type: belong,
		unique: true
	}
}, {
	versionKey: false,
	timestamps: true
})

export default model<IProblem>('problem', schema)
