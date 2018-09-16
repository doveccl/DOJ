import { model, Document, Schema } from 'mongoose'

export enum Status {
	WAIT, // Pending
	AC, // Accepted
	WA, // Wrong Answer
	TLE, // Time Limit Exceed
	MLE, // Memory Limit Exceed
	RE, // Runtime Error
	CE, // Compile Error
	SE, // System Error
	OTHER // Others
}

export interface IResult {
	time: number
	memory: number
	status: Status
}

export interface ISubmission extends Document {
	uid: Schema.Types.ObjectId
	pid: Schema.Types.ObjectId
	cid?: Schema.Types.ObjectId
	code: string
	language: number
	open: boolean
	result: IResult
	cases: IResult[]
	createdAt: Date
	updatedAt: Date
}

const result = new Schema({
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
}, {
	_id: false
})

const schema = new Schema({
	uid: {
		type: Schema.Types.ObjectId,
		required: true,
		index: true
	},
	pid: {
		type: Schema.Types.ObjectId,
		required: true,
		index: true
	},
	cid: {
		type: Schema.Types.ObjectId,
		index: true
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
		default: false
	},
	result: {
		type: result,
		default: {
			time: 0,
			memory: 0,
			status: Status.WAIT
		}
	},
	cases: {
		type: [result],
		default: []
	}
}, {
	versionKey: false,
	timestamps: true
})

export const Submission = model<ISubmission>('submission', schema)
