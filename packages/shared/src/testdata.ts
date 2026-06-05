import { inflateRawSync } from 'node:zlib'
import type { ProblemTestCase } from './judge'

interface ZipEntry {
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
