import {
  CreateBucketCommand,
  GetObjectCommand,
  HeadBucketCommand,
  PutObjectCommand,
  S3Client
} from '@aws-sdk/client-s3'
import { storageConfig } from './config'

export const s3 = new S3Client({
  endpoint: storageConfig.endpoint,
  region: storageConfig.region,
  forcePathStyle: true,
  credentials: {
    accessKeyId: storageConfig.accessKeyId,
    secretAccessKey: storageConfig.secretAccessKey
  }
})

export async function ensureBucket(bucket = storageConfig.bucket) {
  try {
    await s3.send(new HeadBucketCommand({ Bucket: bucket }))
  } catch {
    await s3.send(new CreateBucketCommand({ Bucket: bucket }))
  }
}

export interface PutObjectInput {
  key: string
  body: Uint8Array | string
  contentType: string
  bucket?: string
}

export async function putObject(input: PutObjectInput) {
  await s3.send(
    new PutObjectCommand({
      Bucket: input.bucket ?? storageConfig.bucket,
      Key: input.key,
      Body: input.body,
      ContentType: input.contentType
    })
  )
}

export async function getObjectBytes(key: string, bucket = storageConfig.bucket) {
  const output = await s3.send(new GetObjectCommand({ Bucket: bucket, Key: key }))
  if (!output.Body) return new Uint8Array()
  return output.Body.transformToByteArray()
}
