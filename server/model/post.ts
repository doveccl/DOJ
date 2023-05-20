import { model, Document, Schema } from 'mongoose'
import { IPost } from '../../common/interface'

export type DPost = IPost<Schema.Types.ObjectId, Date> & Document

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

export const Post = model<DPost>('post', schema)
