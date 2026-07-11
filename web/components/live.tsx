import { useEffect } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'

import { api, apiData, apiUrl } from '../client'
import { useSession } from '../session'

const liveKeys = new Set([
  'admin',
  'assignment',
  'assignments',
  'contest',
  'contests',
  'home',
  'problem',
  'problem-state',
  'problems',
  'rank',
  'submission',
  'submissions',
  'user'
])

export function LiveEvents() {
  const client = useQueryClient()
  const session = useSession()
  const site = useQuery({ queryKey: ['site'], queryFn: () => apiData(api.GET('/api/site')) })
  const enabled = session.signedIn || site.data?.allowGuestAccess === true

  useEffect(() => {
    if (!enabled || typeof EventSource === 'undefined') {
      return
    }

    const source = new EventSource(apiUrl('/api/events').toString(), { withCredentials: true })
    let refreshTimer: ReturnType<typeof setTimeout> | undefined
    source.addEventListener('submission', () => {
      if (refreshTimer !== undefined) {
        return
      }
      refreshTimer = setTimeout(() => {
        refreshTimer = undefined
        void client.invalidateQueries({
          predicate: (query) => liveKeys.has(String(query.queryKey[0]))
        })
      }, 300)
    })
    source.addEventListener('submission-progress', () => {
      void client.invalidateQueries({ queryKey: ['submission'] })
    })
    return () => {
      source.close()
      if (refreshTimer !== undefined) {
        clearTimeout(refreshTimer)
      }
    }
  }, [client, enabled])

  return null
}
