import type { JudgeAgentPackageFile } from '@doj/shared/agent'
import { getObjectBytes } from '@doj/shared/storage'

const defaultMaxBytes = 512 * 1024 * 1024

export interface PackageFileCacheOptions {
  maxBytes?: number
  fetchBytes?: (key: string, bucket: string) => Promise<Uint8Array>
}

interface CacheEntry {
  bytes: Uint8Array
  sizeBytes: number
}

export class PackageFileCache {
  private readonly maxBytes: number
  private readonly fetchBytes: (key: string, bucket: string) => Promise<Uint8Array>
  private readonly entries = new Map<string, CacheEntry>()
  private readonly inFlight = new Map<string, Promise<Uint8Array>>()
  private totalBytes = 0

  constructor(options: PackageFileCacheOptions = {}) {
    this.maxBytes = Math.max(0, options.maxBytes ?? defaultMaxBytes)
    this.fetchBytes = options.fetchBytes ?? getObjectBytes
  }

  async get(file: JudgeAgentPackageFile) {
    const key = cacheKey(file)
    const cached = this.entries.get(key)
    if (cached) {
      this.entries.delete(key)
      this.entries.set(key, cached)
      return cached.bytes
    }

    const existing = this.inFlight.get(key)
    if (existing) return existing

    const pending = this.fetchAndStore(file, key)
    this.inFlight.set(key, pending)
    try {
      return await pending
    } finally {
      this.inFlight.delete(key)
    }
  }

  snapshot() {
    return {
      entries: this.entries.size,
      totalBytes: this.totalBytes,
      maxBytes: this.maxBytes
    }
  }

  private async fetchAndStore(file: JudgeAgentPackageFile, key: string) {
    const bytes = await this.fetchBytes(file.objectKey, file.bucket)
    const sizeBytes = file.sizeBytes ?? bytes.byteLength
    if (this.maxBytes > 0 && sizeBytes <= this.maxBytes) {
      const previous = this.entries.get(key)
      if (previous) this.totalBytes -= previous.sizeBytes
      this.entries.delete(key)
      this.entries.set(key, { bytes, sizeBytes })
      this.totalBytes += sizeBytes
      this.evict()
    }
    return bytes
  }

  private evict() {
    for (const [key, entry] of this.entries) {
      if (this.totalBytes <= this.maxBytes) return
      this.entries.delete(key)
      this.totalBytes -= entry.sizeBytes
    }
  }
}

function cacheKey(file: JudgeAgentPackageFile) {
  return `${file.bucket}/${file.objectKey}`
}
