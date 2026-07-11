import { QueryClient, QueryObserver } from '@tanstack/react-query'
import { describe, expect, it } from 'vitest'

import type { Me } from './client'
import { setViewer } from './session'

describe('viewer cache boundary', () => {
  it('clears active and inactive queries before switching users', () => {
    const client = new QueryClient()
    client.setQueryData(['me'], me(1, 'alice'))
    client.setQueryData(['submission', 1], { code: 'alice secret' })
    client.setQueryData(['admin-users'], ['alice'])
    const observer = new QueryObserver(client, {
      queryKey: ['submission', 1],
      queryFn: async () => ({ code: 'bob secret' })
    })
    const unsubscribe = observer.subscribe(() => undefined)

    setViewer(client, me(2, 'bob'))

    expect(observer.getCurrentResult().data).toBeUndefined()
    expect(client.getQueryData(['admin-users'])).toBeUndefined()
    expect(client.getQueryData<Me>(['me'])?.name).toBe('bob')
    unsubscribe()
  })
})

function me(id: number, name: string): Me {
  return { id, name, mail: '', bio: '', avatar: '', admin: false }
}
