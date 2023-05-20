import { model, Document, Schema } from 'mongoose'
import { ISubmission, Status } from '../../common/interface'

export type DSubmission = ISubmission<Schema.Types.ObjectId, Date> & Document

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
  },
  extra: {
    type: String,
    required: false
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

export const Submission = model<DSubmission>('submission', schema)
