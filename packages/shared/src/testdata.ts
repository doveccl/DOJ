import { inflateRawSync } from 'node:zlib'
import type { ProblemTestCase } from './judge'

export interface ZipEntry {
  name: string
  bytes: Uint8Array
}

export interface ParseZipTestCasesOptions {
  maxEntries?: number
  maxEntryBytes?: number
  maxTotalBytes?: number
}

const defaultZipLimits = {
  maxEntries: 240,
  maxEntryBytes: 2 * 1024 * 1024,
  maxTotalBytes: 32 * 1024 * 1024
}

export function parseZipTestCases(
  bytes: Uint8Array,
  options: ParseZipTestCasesOptions = {}
): ProblemTestCase[] {
  const limits = { ...defaultZipLimits, ...options }
  const entries = readZipEntries(bytes, limits)
  return buildCasesFromEntries(entries)
}

// Build cases from already-extracted files (e.g. loose uploads, not a ZIP).
export function parseLooseTestCases(entries: ZipEntry[]): ProblemTestCase[] {
  return buildCasesFromEntries(entries)
}

function buildCasesFromEntries(entries: ZipEntry[]): ProblemTestCase[] {
  const inputs = new Map<string, ZipEntry>()
  const outputs = new Map<string, ZipEntry>()
  const decoder = new TextDecoder()

  for (const entry of entries) {
    const classified = classifyCaseEntry(entry)
    if (!classified) continue
    if (classified.kind === 'input') inputs.set(classified.key, entry)
    if (classified.kind === 'output') outputs.set(classified.key, entry)
  }

  return [...inputs.keys()].sort(compareCaseKey).flatMap((key) => {
    const input = inputs.get(key)
    const output = outputs.get(key)
    if (!input || !output) return []

    return [
      {
        name: key,
        input: decoder.decode(input.bytes),
        output: decoder.decode(output.bytes),
        hidden: true
      }
    ]
  })
}

function readZipEntries(bytes: Uint8Array, limits: Required<ParseZipTestCasesOptions>) {
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
  const entries: ZipEntry[] = []
  let offset = 0
  let totalBytes = 0

  while (offset + 30 <= bytes.length) {
    const signature = view.getUint32(offset, true)
    if (signature !== 0x04034b50) break

    const flags = view.getUint16(offset + 6, true)
    const method = view.getUint16(offset + 8, true)
    const compressedSize = view.getUint32(offset + 18, true)
    const filenameLength = view.getUint16(offset + 26, true)
    const extraLength = view.getUint16(offset + 28, true)
    const dataStart = offset + 30 + filenameLength + extraLength
    const dataEnd = dataStart + compressedSize
    if (flags & 0x08) {
      throw new Error(
        'ZIP data descriptors are not supported; create archives with known entry sizes.'
      )
    }
    if (dataEnd > bytes.length) throw new Error('Invalid ZIP entry size.')

    const name = new TextDecoder().decode(bytes.slice(offset + 30, offset + 30 + filenameLength))
    if (!name.endsWith('/')) {
      if (entries.length >= limits.maxEntries) throw new Error('ZIP contains too many files.')
      const decoded = decodeEntry(method, bytes.slice(dataStart, dataEnd))
      if (decoded.byteLength > limits.maxEntryBytes) {
        throw new Error(`ZIP entry is too large: ${name}`)
      }
      totalBytes += decoded.byteLength
      if (totalBytes > limits.maxTotalBytes) {
        throw new Error('ZIP uncompressed data is too large.')
      }
      entries.push({
        name,
        bytes: decoded
      })
    }

    offset = dataEnd
  }

  return entries
}

function decodeEntry(method: number, bytes: Uint8Array) {
  if (method === 0) return bytes
  if (method === 8) return inflateRawSync(bytes)
  throw new Error(`Unsupported ZIP compression method: ${method}`)
}

function normalizeName(name: string) {
  return name.replace(/\\/g, '/').replace(/^.*\//, '')
}

function classifyCaseEntry(entry: ZipEntry) {
  const name = normalizeName(entry.name)
  const lower = name.toLowerCase()
  const stem = lower.replace(/\.[^.]+$/, '')
  const extension = lower.includes('.') ? lower.replace(/^.*\./, '.') : ''
  const kind =
    extension === '.in' || /input|(^|[^a-z])in([^a-z]|$)/.test(stem)
      ? 'input'
      : extension === '.out' ||
          extension === '.ans' ||
          /output|answer|ans|(^|[^a-z])out([^a-z]|$)/.test(stem)
        ? 'output'
        : null
  if (!kind) return null

  const number = stem.match(/\d+/g)?.at(-1)
  const stripped =
    stem
      .replace(/input|output|answer|ans/g, '')
      .replace(/(^|[^a-z])in([^a-z]|$)/g, '$1$2')
      .replace(/(^|[^a-z])out([^a-z]|$)/g, '$1$2')
      .replace(/^[\s._-]+|[\s._-]+$/g, '') || stem
  const key = number ? String(Number.parseInt(number, 10)) : stripped

  return { kind, key }
}

function compareCaseKey(left: string, right: string) {
  const leftNumber = Number.parseInt(left, 10)
  const rightNumber = Number.parseInt(right, 10)
  if (Number.isFinite(leftNumber) && Number.isFinite(rightNumber)) return leftNumber - rightNumber
  return left.localeCompare(right)
}

// Build a stored (uncompressed, method 0) ZIP from loose entries so that loose
// testdata uploads share the exact same storage + agent parsing path as ZIPs.
export function buildStoredZip(entries: ZipEntry[]): Uint8Array {
  const encoder = new TextEncoder()
  const localParts: Uint8Array[] = []
  const centralParts: Uint8Array[] = []
  let offset = 0

  for (const entry of entries) {
    const nameBytes = encoder.encode(entry.name)
    const crc = crc32(entry.bytes)
    const size = entry.bytes.byteLength

    const local = new Uint8Array(30 + nameBytes.length)
    const lv = new DataView(local.buffer)
    lv.setUint32(0, 0x04034b50, true)
    lv.setUint16(4, 20, true)
    lv.setUint16(6, 0, true)
    lv.setUint16(8, 0, true) // method 0 (stored)
    lv.setUint16(10, 0, true)
    lv.setUint16(12, 0, true)
    lv.setUint32(14, crc, true)
    lv.setUint32(18, size, true)
    lv.setUint32(22, size, true)
    lv.setUint16(26, nameBytes.length, true)
    lv.setUint16(28, 0, true)
    local.set(nameBytes, 30)
    localParts.push(local, entry.bytes)

    const central = new Uint8Array(46 + nameBytes.length)
    const cv = new DataView(central.buffer)
    cv.setUint32(0, 0x02014b50, true)
    cv.setUint16(4, 20, true)
    cv.setUint16(6, 20, true)
    cv.setUint16(8, 0, true)
    cv.setUint16(10, 0, true)
    cv.setUint16(12, 0, true)
    cv.setUint16(14, 0, true)
    cv.setUint32(16, crc, true)
    cv.setUint32(20, size, true)
    cv.setUint32(24, size, true)
    cv.setUint16(28, nameBytes.length, true)
    cv.setUint32(42, offset, true)
    central.set(nameBytes, 46)
    centralParts.push(central)

    offset += local.length + size
  }

  const centralSize = centralParts.reduce((sum, part) => sum + part.length, 0)
  const end = new Uint8Array(22)
  const ev = new DataView(end.buffer)
  ev.setUint32(0, 0x06054b50, true)
  ev.setUint16(8, entries.length, true)
  ev.setUint16(10, entries.length, true)
  ev.setUint32(12, centralSize, true)
  ev.setUint32(16, offset, true)

  const total = offset + centralSize + end.length
  const out = new Uint8Array(total)
  let cursor = 0
  for (const part of [...localParts, ...centralParts, end]) {
    out.set(part, cursor)
    cursor += part.length
  }
  return out
}

const crcTable = (() => {
  const table = new Uint32Array(256)
  for (let n = 0; n < 256; n += 1) {
    let c = n
    for (let k = 0; k < 8; k += 1) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
    table[n] = c >>> 0
  }
  return table
})()

function crc32(bytes: Uint8Array) {
  let crc = 0xffffffff
  for (let i = 0; i < bytes.length; i += 1) {
    crc = crcTable[(crc ^ bytes[i]) & 0xff] ^ (crc >>> 8)
  }
  return (crc ^ 0xffffffff) >>> 0
}
