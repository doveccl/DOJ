import { describe, expect, it } from 'vitest'

import { dataPairRows } from './files'

describe('dataPairRows', () => {
  it('keeps duplicate case sides visible', () => {
    const rows = dataPairRows([
      { key: 'data/1.in', name: '1.in', size: 1 },
      { key: 'data/input1.txt', name: 'input1.txt', size: 1 },
      { key: 'data/1.out', name: '1.out', size: 1 }
    ])
    expect(rows).toHaveLength(2)
    expect(rows[0]).toMatchObject({ key: '1', input: { name: '1.in' }, output: { name: '1.out' } })
    expect(rows[1]).toMatchObject({ key: 'data/input1.txt', input: { name: 'input1.txt' } })
  })
})
