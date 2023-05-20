import { User } from './user'
import { Config } from './config'
import { hashSync } from 'bcryptjs'
import { Group } from '../../common/interface'

export function initDB() {
  User.findOne().then(c => c || User.create({
    name: 'admin',
    mail: 'admin@d.oj',
    group: Group.root,
    password: hashSync('admin')
  }))

  Config.findById('notification').then(c => c || Config.create({
    _id: 'notification',
    value: 'DOJ notification'
  }))
}
