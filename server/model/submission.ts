import { Schema, Document, model } from 'mongoose'

export enum Status {
	AC, // Accepted
	WA, // Wrong Answer
	TLE, // Time Limit Exceed
	MLE, // Memory Limit Exceed
	RE, // Runtime Error
	CE, // Compile Error
	SE, // System Error
	WAIT, // Pending
	OTHER // Others
}

export interface IResult {
	time: number
	memory: number
	status: Status
}

export interface ISubmission extends Document {
	user: Schema.Types.ObjectId
	problem: Schema.Types.ObjectId
	contest?: Schema.Types.ObjectId
	code: string
	language: number
	open: boolean
	result: IResult
	cases: IResult[]
}

const result = {
	time: {
		type: Number,
		required: true
	},
	memory: {
		type: Number,
		required: true
	},
	status: {
		type: Number,
		required: true
	}
}

const schema = new Schema({
	user: {
		type: Schema.Types.ObjectId,
		ref: 'user',
		required: true
	},
	problem: {
		type: Schema.Types.ObjectId,
		ref: 'problem',
		required: true
	},
	contest: {
		type: Schema.Types.ObjectId,
		ref: 'contest',
		required: false
	},
	code: {
		type: String,
		required: true,
		maxlength: 100 * 1024
	},
	language: {
		type: Number,
		required: true
	},
	open: {
		type: Boolean,
		required: false,
		default: false
	},
	result: {
		type: result,
		required: false,
		default: {
			time: 0,
			memory: 0,
			status: Status.WAIT
		}
	},
	cases: {
		type: [result],
		required: false,
		default: <IResult[]>[]
	}
}, {
	versionKey: false,
	timestamps: true
})

export default model<ISubmission>('submission', schema)
