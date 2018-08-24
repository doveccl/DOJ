import { Schema, Document, model } from 'mongoose'

export enum ContestType { OI, ICPC }
export enum Status { NONE, OK, FAIL }

export class PResult {
	public score = 0
	public count = 0
	public status = Status.NONE
}

export class UResult {
	public name: string
	public score = 0
	public solve = 0
	public penalty = 0
	public result: PResult[]
	constructor(name: string, count: number) {
		this.name = name
		this.result = []
		for (let cnt = count; cnt--; ) {
			this.result.push(new PResult())
		}
	}
}

export interface IContest extends Document {
	title: string
	description: string
	type: ContestType
	startAt: Date
	endAt: Date
	problems: Schema.Types.ObjectId[]
	result: UResult[]
}

const schema = new Schema({
	title: {
		type: String,
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
		require: true
	},
	problems: [{
		type: Schema.Types.ObjectId,
		ref: 'problem'
	}],
	result: {
		type: [UResult],
		default: []
	}
}, {
	versionKey: false,
	timestamps: true
})

export default model<IContest>('contest', schema)
