import { CreateBucketCommand, HeadBucketCommand, S3Client as AwsS3Client } from '@aws-sdk/client-s3'
import { S3Client as BunS3Client } from 'bun'
import { storageConfig } from './config'

const credentials = {
  accessKeyId: storageConfig.accessKeyId,
  secretAccessKey: storageConfig.secretAccessKey,
  region: storageConfig.region,
  endpoint: storageConfig.endpoint,
  bucket: storageConfig.bucket
}

export const s3 = new BunS3Client(credentials)

const bucketAdmin = new AwsS3Client({
  endpoint: storageConfig.endpoint,
  region: storageConfig.region,
  forcePathStyle: true,
  credentials: {
    accessKeyId: storageConfig.accessKeyId,
    secretAccessKey: storageConfig.secretAccessKey
  }
})

export async function ensureBucket(bucket = storageConfig.bucket) {
  // Bun's native S3 client is object-focused; bucket provisioning still needs the AWS-compatible admin API.
  try {
    await bucketAdmin.send(new HeadBucketCommand({ Bucket: bucket }))
  } catch {
    await bucketAdmin.send(new CreateBucketCommand({ Bucket: bucket }))
  }
}

export interface PutObjectInput {
  key: string
  body: Uint8Array | string
  contentType: string
  bucket?: string
}

export async function putObject(input: PutObjectInput) {
  const client =
    input.bucket && input.bucket !== storageConfig.bucket
      ? new BunS3Client({ ...credentials, bucket: input.bucket })
      : s3
  await client.write(input.key, input.body, { type: input.contentType })
}

export async function getObjectBytes(key: string, bucket = storageConfig.bucket) {
  const client = bucket !== storageConfig.bucket ? new BunS3Client({ ...credentials, bucket }) : s3
  return client.file(key).bytes()
}
