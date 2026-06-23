import { useEffect } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'

import { apiUrl, getSite } from '../client'
import { useSession } from '../session'

const liveKeys = new Set([
  'admin',
  'assignment',
  'assignments',
  'contest',
  'contests',
  'home',
  'problem',
  'problems',
  'rank',
  'submission',
  'submissions',
  'user'
])

export function LiveEvents() {
  const client = useQueryClient()
  const session = useSession()
  const site = useQuery({ queryKey: ['site'], queryFn: getSite })
  const enabled = session.signedIn || site.data?.guest === true

  useEffect(() => {
    if (!enabled || typeof EventSource === 'undefined') {
      return
    }

    const source = new EventSource(apiUrl('/api/events').toString(), { withCredentials: true })
    source.addEventListener('submission', () => {
      void client.invalidateQueries({
        predicate: (query) => liveKeys.has(String(query.queryKey[0]))
      })
    })
    return () => source.close()
  }, [client, enabled])

  return null
}
