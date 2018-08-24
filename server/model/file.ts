import * as fs from 'fs'
import { Schema, Document, model, connection } from 'mongoose'
import { ObjectID, GridFSBucket, GridFSBucketOpenUploadStreamOptions } from 'mongodb'

export const TYPE_REG = /(:?image|pdf|zip)/

export interface IFile extends Document {
	filename: string
	contentType: string
	length: number
	chunkSize: number
	uploadDate: Date
	metadata: any
	aliases: string[]
	md5: string
}

const schema = new Schema({
	filename: String,
	contentType: String,
	length: Number,
	chunkSize: Number,
	uploadDate: Date,
	metadata: Schema.Types.Mixed,
	aliases: [String],
	md5: String
}, {
	versionKey: false
})

/**
 * mongoose model for GridFS
 * do not creat/delete file with this model
 * do not update file content with this model
 */
const FS = model<IFile>('fs.file', schema)

/**
 * mongodb GridFS Bucket
 */
let bucket: GridFSBucket
connection.on('open', () => bucket = new GridFSBucket(connection.db))

export default class {
	public static create(
		path: string, filename: string,
		opts?: GridFSBucketOpenUploadStreamOptions
	) {
		return new Promise<any>((resolve, reject) => {
			const stream = bucket.openUploadStream(filename, opts)
			fs.createReadStream(path).pipe(stream)
				.on('error', error => reject(error))
				.on('finish', () => resolve(stream.id))
		})
	}
	public static creatReadStream(id: any) {
		return bucket.openDownloadStream(new ObjectID(id))
	}
	public static countDocuments(conditions?: any) {
		return FS.countDocuments(conditions)
	}
	public static find(conditions?: any) {
		return FS.find(conditions)
	}
	public static findById(id: any) {
		return FS.findById(id)
	}
	public static findByIdAndRemove(id: any) {
		return new Promise<any>((resolve, reject) => {
			bucket.delete(new ObjectID(id), error => {
				if (error) { reject(error) }
				resolve({ id })
			})
		})
	}
}
