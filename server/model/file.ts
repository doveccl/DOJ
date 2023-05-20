import fs from 'fs'
import { GridFSBucket, GridFSBucketWriteStreamOptions, ObjectId } from 'mongodb'
import { connection, model, Document, Schema } from 'mongoose'
import { IFile } from '../../common/interface'

export type DFile = IFile<Schema.Types.ObjectId, Date> & Document

const schema = new Schema({
  filename: String,
  length: Number,
  chunkSize: Number,
  uploadDate: Date,
  metadata: Schema.Types.Mixed
}, {
  versionKey: false
})

/**
 * mongoose model for GridFS
 * do not creat/delete file with this model
 * do not update file content with this model
 */
const FS = model<DFile>('fs.file', schema)

/**
 * mongodb GridFS Bucket
 */
let bucket: GridFSBucket
connection.on('open', () => bucket = new GridFSBucket(connection.db))

export const TYPE_REG = /(:?image|pdf|zip)/

export class File {
  public static create(
    path: string, filename: string,
    opts?: GridFSBucketWriteStreamOptions
  ) {
    return new Promise<ObjectId>((resolve, reject) => {
      const stream = bucket.openUploadStream(filename, opts)
      stream.on('error', error => reject(error))
      stream.on('finish', () => resolve(stream.id))
      fs.createReadStream(path).pipe(stream)
    })
  }
  public static creatReadStream(id: any) {
    return bucket.openDownloadStream(new ObjectId(id))
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
  public static findByIdAndUpdate(id: any, data: any) {
    return FS.findByIdAndUpdate(id, data)
  }
  public static findByIdAndRemove(id: any) {
    return bucket.delete(new ObjectId(id)).then(() => ({ id }))
  }
}
