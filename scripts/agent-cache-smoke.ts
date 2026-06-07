import { PackageFileCache } from '../apps/agent/src/package-cache'

const fetched: string[] = []
const cache = new PackageFileCache({
  maxBytes: 6,
  fetchBytes: async (key, bucket) => {
    fetched.push(`${bucket}/${key}`)
    return new TextEncoder().encode(key)
  }
})

await cache.get({ path: 'data/1.in', bucket: 'doj', objectKey: 'one', sizeBytes: 3 })
await cache.get({ path: 'data/1.in', bucket: 'doj', objectKey: 'one', sizeBytes: 3 })

if (fetched.length !== 1) {
  throw new Error(`expected cache hit for repeated object, got fetches: ${fetched.join(', ')}`)
}

await cache.get({ path: 'data/2.in', bucket: 'doj', objectKey: 'two', sizeBytes: 3 })
await cache.get({ path: 'data/3.in', bucket: 'doj', objectKey: 'three', sizeBytes: 5 })
await cache.get({ path: 'data/1.in', bucket: 'doj', objectKey: 'one', sizeBytes: 3 })

if (fetched.join(',') !== 'doj/one,doj/two,doj/three,doj/one') {
  throw new Error(`expected LRU eviction, got fetches: ${fetched.join(',')}`)
}

const disabled = new PackageFileCache({
  maxBytes: 0,
  fetchBytes: async (key, bucket) => {
    fetched.push(`disabled:${bucket}/${key}`)
    return new TextEncoder().encode(key)
  }
})
await disabled.get({ path: 'data/4.in', bucket: 'doj', objectKey: 'four', sizeBytes: 4 })
await disabled.get({ path: 'data/4.in', bucket: 'doj', objectKey: 'four', sizeBytes: 4 })

if (fetched.filter((item) => item === 'disabled:doj/four').length !== 2) {
  throw new Error('expected disabled cache to fetch every time')
}

console.log({
  fetches: fetched,
  snapshot: cache.snapshot()
})
