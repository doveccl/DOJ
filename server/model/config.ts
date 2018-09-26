import { model, Document, Schema } from 'mongoose'
import { IConfig } from '../../common/interface'

export type DConfig = IConfig & Document

const schema = new Schema({
	_id: {
		type: String,
		required: true
	},
	value: {
		type: Schema.Types.Mixed,
		required: true
	}
}, {
	versionKey: false
})

export const Config = model<DConfig>('config', schema)

const createIfNotExist = async (config: Partial<DConfig>) => {
	if (!await Config.findById(config._id)) {
		await Config.create(config)
	}
}

createIfNotExist({
	_id: 'notification',
	value: 'DOJ notification'
})
