import type { PackageFile } from '../../../client'

export type DataPairRow = { key: string; input?: PackageFile; output?: PackageFile }

export function dataPairRows(files: PackageFile[]) {
  const rows: DataPairRow[] = []
  const byStem = new Map<string, DataPairRow>()
  for (const file of files) {
    const { stem, kind } = dataCaseStem(file.name)
    if (stem === '' || kind === '') {
      rows.push({ key: file.key, input: file })
      continue
    }
    let row = byStem.get(stem)
    if (!row) {
      row = { key: stem }
      byStem.set(stem, row)
      rows.push(row)
    }
    if (kind === 'in') {
      if (row.input) {
        rows.push({ key: file.key, input: file })
        continue
      }
      row.input = file
    } else {
      if (row.output) {
        rows.push({ key: file.key, output: file })
        continue
      }
      row.output = file
    }
  }
  return rows
}

export function acceptedDataFile(name: string) {
  return ['.in', '.out', '.ans', '.txt'].some((suffix) => name.toLowerCase().endsWith(suffix))
}

export function downloadURL(url: string, filename: string) {
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  document.body.appendChild(anchor)
  anchor.click()
  anchor.remove()
}

function dataCaseStem(name: string) {
  const base = name.split('/').pop() ?? name
  const lower = base.toLowerCase()
  const stem = base.match(/\d+/)?.[0]
  if (stem) {
    if (lower.includes('in')) {
      return { stem, kind: 'in' }
    }
    if (lower.includes('out') || lower.includes('ans')) {
      return { stem, kind: 'out' }
    }
  }
  const index = base.lastIndexOf('.')
  if (index <= 0) {
    return { stem: '', kind: '' }
  }
  switch (lower.slice(index)) {
    case '.in':
      return { stem: base.slice(0, index), kind: 'in' }
    case '.out':
    case '.ans':
      return { stem: base.slice(0, index), kind: 'out' }
    default:
      return { stem: '', kind: '' }
  }
}
