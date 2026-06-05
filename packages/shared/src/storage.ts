import { S3Client as BunS3Client } from 'bun'
import { createHash, createHmac } from 'node:crypto'

export const storageConfig = {
  endpoint: process.env.S3_ENDPOINT ?? 'http://localhost:9000',
  region: process.env.S3_REGION ?? 'us-east-1',
  accessKeyId: process.env.S3_ACCESS_KEY_ID ?? 'minioadmin',
  secretAccessKey: process.env.S3_SECRET_ACCESS_KEY ?? 'minioadmin',
  bucket: process.env.S3_BUCKET ?? 'doj'
}

const credentials = {
  accessKeyId: storageConfig.accessKeyId,
  secretAccessKey: storageConfig.secretAccessKey,
  region: storageConfig.region,
  endpoint: storageConfig.endpoint,
  bucket: storageConfig.bucket
}

const s3 = new BunS3Client(credentials)

export interface PutObjectInput {
  key: string
  body: Uint8Array | string
  contentType: string
  bucket?: string
}

export async function ensureBucket(bucket = storageConfig.bucket) {
  const head = await signedBucketRequest('HEAD', bucket)
  if (head.ok) return
  if (![403, 404].includes(head.status)) {
    throw new Error(`S3 bucket check failed: ${head.status} ${await head.text()}`)
  }

  const put = await signedBucketRequest('PUT', bucket)
  if (!put.ok && put.status !== 409) {
    throw new Error(`S3 bucket create failed: ${put.status} ${await put.text()}`)
  }
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

async function signedBucketRequest(method: 'HEAD' | 'PUT', bucket: string) {
  const endpoint = new URL(storageConfig.endpoint)
  const path = `/${encodeURIComponent(bucket)}`
  const url = new URL(path, endpoint)
  const now = new Date()
  const amzDate = toAmzDate(now)
  const dateStamp = amzDate.slice(0, 8)
  const payloadHash = sha256Hex('')
  const headers = {
    host: url.host,
    'x-amz-content-sha256': payloadHash,
    'x-amz-date': amzDate
  }
  const signedHeaders = Object.keys(headers).sort().join(';')
  const canonicalHeaders = Object.entries(headers)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, value]) => `${key}:${value}\n`)
    .join('')
  const canonicalRequest = [method, path, '', canonicalHeaders, signedHeaders, payloadHash].join(
    '\n'
  )
  const scope = `${dateStamp}/${storageConfig.region}/s3/aws4_request`
  const stringToSign = ['AWS4-HMAC-SHA256', amzDate, scope, sha256Hex(canonicalRequest)].join('\n')
  const signature = hmacHex(signingKey(dateStamp), stringToSign)

  return fetch(url, {
    method,
    headers: {
      ...headers,
      authorization: [
        `AWS4-HMAC-SHA256 Credential=${storageConfig.accessKeyId}/${scope}`,
        `SignedHeaders=${signedHeaders}`,
        `Signature=${signature}`
      ].join(', ')
    }
  })
}

function signingKey(dateStamp: string) {
  const date = hmac(`AWS4${storageConfig.secretAccessKey}`, dateStamp)
  const region = hmac(date, storageConfig.region)
  const service = hmac(region, 's3')
  return hmac(service, 'aws4_request')
}

function hmac(key: string | Buffer, value: string) {
  return createHmac('sha256', key).update(value).digest()
}

function hmacHex(key: Buffer, value: string) {
  return createHmac('sha256', key).update(value).digest('hex')
}

function sha256Hex(value: string) {
  return createHash('sha256').update(value).digest('hex')
}

function toAmzDate(date: Date) {
  return date.toISOString().replace(/[:-]|\.\d{3}/g, '')
}
