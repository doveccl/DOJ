const required = ['DATABASE_URL']

for (const key of required) {
  if (!process.env[key]) {
    console.warn(`${key} is not set; using development defaults where available.`)
  }
}

console.log('Environment check complete.')
