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

const createIfNotExist = async (id: string, value: any) => {
	if (!await Config.findById(id)) {
		await Config.create({ _id: id, value })
	}
}

createIfNotExist('notification', 'DOJ notification')
