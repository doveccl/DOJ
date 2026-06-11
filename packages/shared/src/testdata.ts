export interface ZipEntry {
  name: string
  bytes: Uint8Array
}

export interface TestdataCase {
  name?: string
  input: string
  output: string
  hidden?: boolean
}

// Pair input/output data files (any naming) into ordered test cases. Used by the
// agent to derive default-mode cases from a problem package's `data/` files.
export function buildCasesFromPackageData(
  files: Array<{ path: string; bytes: Uint8Array }>
): TestdataCase[] {
  return buildCasesFromEntries(files.map((file) => ({ name: file.path, bytes: file.bytes })))
}

function buildCasesFromEntries(entries: ZipEntry[]): TestdataCase[] {
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
