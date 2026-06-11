const required = ['DATABASE', 'S3_ENDPOINT', 'S3_ACCESS_KEY', 'S3_SECRET_KEY', 'SECRET']

for (const key of required) {
  if (!process.env[key]) {
    throw new Error(`${key} is required`)
  }
}

console.log('Environment check complete.')
