import { model, Document, Schema } from 'mongoose'

export interface IConfig extends Document {
	_id: string
	value: any
}

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

const modelConfig = model<IConfig>('config', schema)

const createIfNotExist = async (id: string, value: any) => {
	if (!await modelConfig.findById(id)) {
		await modelConfig.create({ _id: id, value })
	}
}

createIfNotExist('notification', 'DOJ notification')
createIfNotExist('faq', 'DOJ FAQ')

export { modelConfig as Config }
