import { Schema, Document, model } from 'mongoose'

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

const createIfNotExist = async (_id: string, value: any) => {
	if (!await modelConfig.findById(_id)) {
		await modelConfig.create({ _id, value })
	}
}

createIfNotExist('notification', 'DOJ notification')
createIfNotExist('faq', 'DOJ FAQ')

export default modelConfig
