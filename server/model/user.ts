import { Schema, Document, model } from 'mongoose'

export interface IUser extends Document {
	name: string;
	mail: string;
	admin: number;
	password: string;
	introduction?: string;
}

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
	admin: {
		type: Number,
		required: true,
		min: 0, default: 0
	},
	password: {
		type: String,
		required: true
	},
	introduction: {
		type: String,
		maxlength: 200,
		required: false
	}
}, {
	versionKey: false,
	timestamps: true
})

export default model<IUser>('User', schema)
