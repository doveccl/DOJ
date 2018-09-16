import { model, Document, Schema } from 'mongoose'

export interface IPost extends Document {
	uid: Schema.Types.ObjectId
	topic: Schema.Types.ObjectId
	content: string
	createdAt: Date
	updatedAt: Date
}

const schema = new Schema({
	uid: {
		type: Schema.Types.ObjectId,
		required: true
	},
	topic: {
		type: Schema.Types.ObjectId,
		required: true,
		index: true
	},
	content: {
		type: String,
		required: true,
		minlength: 5,
		maxlength: 100 * 1024
	}
}, {
	versionKey: false,
	timestamps: true
})

export const Post = model<IPost>('post', schema)
