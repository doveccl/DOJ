import { inflateRawSync } from 'node:zlib'
import type { ProblemTestCase } from './judge'

interface ZipEntry {
  name: string
  bytes: Uint8Array
}

export function parseZipTestCases(bytes: Uint8Array): ProblemTestCase[] {
  const entries = readZipEntries(bytes)
  const byName = new Map(entries.map((entry) => [normalizeName(entry.name), entry.bytes]))
  const inputNames = [...byName.keys()]
    .filter((name) => name.endsWith('.in'))
    .sort((left, right) => compareCaseName(left, right))
  const decoder = new TextDecoder()

  return inputNames.flatMap((inputName) => {
    const base = inputName.slice(0, -'.in'.length)
    const output = byName.get(`${base}.out`) ?? byName.get(`${base}.ans`)
    const input = byName.get(inputName)
    if (!input || !output) return []

    return [
      {
        name: base,
        input: decoder.decode(input),
        output: decoder.decode(output),
        hidden: true
      }
    ]
  })
}

function readZipEntries(bytes: Uint8Array) {
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
  const entries: ZipEntry[] = []
  let offset = 0

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
      entries.push({
        name,
        bytes: decodeEntry(method, bytes.slice(dataStart, dataEnd))
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

function compareCaseName(left: string, right: string) {
  const leftNumber = Number.parseInt(left, 10)
  const rightNumber = Number.parseInt(right, 10)
  if (Number.isFinite(leftNumber) && Number.isFinite(rightNumber)) return leftNumber - rightNumber
  return left.localeCompare(right)
}
