const Schema = require('mongoose').Schema
const ObjectId = Schema.Types.ObjectId

exports.user = new Schema({
  name: {
    type: String,
    unique: true,
    match: /^[a-zA-Z0-9][a-zA-Z0-9_]{2,14}$/
  },
  mail: {
    type: String,
    unique: true,
    match: /^[\w.]+@(?:[a-z0-9]+(?:-[a-z0-9]+)*\.)+[a-z]{2,}$/
  },
  admin: Number,
  password: String,
  introduction: String,
  join_time: {
    type: Date,
    default: Date.now
  }
})

exports.problem = new Schema({
  title: String,
  content: String,
  enable: {
    type: Boolean,
    default: true
  },
  data: ObjectId
})

exports.contest = new Schema({
  name: String,
  instruction: String,
  start_time: Date,
  end_time: Date,
  problems: [ObjectId],
  type: {
    type: String,
    match: /^(?:oi|icpc)$/
  }
})

const point = new Schema({
  status: String,
  time: Number,
  memory: Number
})
exports.submission = new Schema({
  user_id: ObjectId,
  problem_id: ObjectId,
  contest_id: {
    type: ObjectId,
    default: null
  },
  submit_time: {
    type: Date,
    default: Date.now
  },
  language: ObjectId,
  code: String,
  result: {
    status: String,
    time: Number,
    memory: Number,
    points: [point]
  }
})

exports.language = new Schema({
  name: String,
  subfix: String,
  config: Object
})
