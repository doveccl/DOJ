export const storageConfig = {
  endpoint: process.env.S3_ENDPOINT ?? 'http://localhost:9000',
  region: process.env.S3_REGION ?? 'us-east-1',
  accessKeyId: process.env.S3_ACCESS_KEY_ID ?? 'doj',
  secretAccessKey: process.env.S3_SECRET_ACCESS_KEY ?? 'doj-password',
  bucket: process.env.S3_BUCKET ?? 'doj'
}
