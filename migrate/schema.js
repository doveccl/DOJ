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
  title: String,
  instruction: String,
  start_time: Date,
  end_time: Date,
  problems: [ObjectId],
  type: {
    type: String,
    match: /^(?:oi|icpc)$/
  }
})

exports.submission = new Schema({
  user_id: ObjectId,
  problem_id: ObjectId,
  contest_id: {
    type: ObjectId,
    default: null
  },
  time: {
    type: Date,
    default: Date.now
  },
  language: ObjectId,
  code: String,
  result: {
    status: {
      type: String,
      default: 'wait'
    },
    time: {
      type: Number,
      default: 0
    },
    memory: {
      type: Number,
      default: 0
    }
  },
  cases: []
})

exports.language = new Schema({
  name: String,
  config: Object
})
