import { Schema, Document, model } from 'mongoose'

export interface IProblem extends Document {
	title: string;
	content: string;
	tags: string[];
	timeLimit: number;
	memoryLimit: number;
	solve: number;
	submit: number;
	data?: Schema.Types.ObjectId;
	contest?: Schema.Types.ObjectId;
}

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
		required: true,
		default: <string[]>[]
	},
	timeLimit: {
		type: Number,
		required: true,
		min: 0, default: 1
	},
	memoryLimit: {
		type: Number,
		required: true,
		min: 0, default: 64
	},
	solve: {
		type: Number,
		required: true,
		min: 0, default: 0
	},
	submit: {
		type: Number,
		required: true,
		min: 0, default: 0
	},
	data: {
		type: Schema.Types.ObjectId,
		ref: 'fs.file',
		required: false
	},
	contest: {
		type: Schema.Types.ObjectId,
		ref: 'contest',
		required: false
	}
}, {
	versionKey: false,
	timestamps: true
})

export default model<IProblem>('Problem', schema)
