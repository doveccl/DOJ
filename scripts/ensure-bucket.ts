import { ensureBucket, putObject, storageConfig } from '../packages/storage/src'

await ensureBucket()
await putObject({
  key: '.healthcheck',
  body: `ok ${new Date().toISOString()}\n`,
  contentType: 'text/plain'
})

console.log(`Ensured S3 bucket: ${storageConfig.bucket}`)
