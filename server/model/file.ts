import * as G from 'gridfs-stream'
import { createReadStream } from 'fs'
import { Schema, model, connection, mongo } from 'mongoose'

// patch for gridfs-stream
eval(`G.prototype.findOne = ${
	G.prototype.findOne.toString().replace('nextObject', 'next')
}`)

interface IFile {
	_id: Schema.Types.ObjectId;
	filename: string;
	contentType: string;
	length: number;
	chunkSize: number;
	uploadDate: Date;
	metadata: any;
	md5: string;
}

interface IFileOption {
	filename?: string;
	content_type?: string;
	metadata?: any;
}

/**
 * Create schema for GridFS
 * 1. populate problem.data with file
 * 2. list files for management
 */
export const FS = model('fs.file', new Schema())

export default class {
	private static grid: G.Grid
	public static init() {
		this.grid = G(connection.db, mongo)
	}
	public static create(path: string, opts?: IFileOption) {
		return new Promise<IFile>((resolve, reject) => {
			const stream = this.grid.createWriteStream(opts)
			createReadStream(path).pipe(stream)
			stream.on('error', error => reject(error))
			stream.on('close', data => resolve(data))
		})
	}
	public static createReadStream(id: any) {
		return this.grid.createReadStream({ _id: id })
	}
	public static findById(id: any) {
		return new Promise<IFile>((resolve, reject) => {
			this.grid.findOne({ _id: id }, (error, file) => {
				if (error) { reject(error) }
				resolve(file)
			})
		})
	}
	public static findByIdAndRemove(id: any) {
		return new Promise<any>((resolve, reject) => {
			this.grid.remove({ _id: id }, error => {
				if (error) { reject(error) }
				resolve({ id })
			})
		})
	}
}
