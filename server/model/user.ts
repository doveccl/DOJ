import { Schema, Document, model } from 'mongoose'
import * as bcrypt from 'bcryptjs'

export interface User {
	name: String;
	mail: String;
	admin: Number;
	password: String;
	introduction?: String;
	join_time: Date;
}

export interface UserModel extends User, Document {
	checkPassword(pwd: String): Boolean;
	checkToken(tkn: String): Boolean;
	getToken(): String;
}

const userSchema = new Schema({
	name: String,
	mail: String,
	admin: Number,
	password: String,
	introduction: String,
  join_time: { type: Date, default: Date.now }
}, {
  versionKey: false
})

userSchema.methods = class {
	public checkPassword(pwd: String) {
		return false
	}
}

export default model<UserModel>('User', userSchema)
