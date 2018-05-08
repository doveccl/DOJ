const Schema = require('mongoose').Schema

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
