import { createContext, useCallback, useContext, useMemo } from 'react'
import type { ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'

import { api, apiData, apiEmpty } from './client'
import type { Me } from './client'

type Role = 'guest' | 'user' | 'admin'

type SessionState = {
  role: Role
  admin: boolean
  signedIn: boolean
  name: string
  avatar?: string
  mustChangePassword: boolean
  refresh: () => Promise<void>
  login: (name: string, password: string) => Promise<void>
  register: (values: { name: string; mail: string; password: string }) => Promise<void>
  logout: () => Promise<void>
  setAvatar: (avatar: string) => void
}

const SessionContext = createContext<SessionState | null>(null)

function getMe() {
  return apiData(api.GET('/api/me'))
}

const guest: Me = {
  id: 0,
  name: '',
  mail: '',
  bio: '',
  avatar: '',
  admin: false,
  mustChangePassword: false
}

export function setViewer(client: QueryClient, next: Me) {
  const current = client.getQueryData<Me>(['me'])
  if ((current?.id ?? 0) !== next.id || (current?.admin ?? false) !== next.admin) {
    const other = (query: { queryKey: readonly unknown[] }) => query.queryKey[0] !== 'me'
    client.removeQueries({ type: 'inactive', predicate: other })
    void client.resetQueries({ type: 'active', predicate: other })
  }
  client.setQueryData(['me'], next)
}

export function SessionProvider({ children }: { children: ReactNode }) {
  const client = useQueryClient()
  const loadMe = useCallback(async () => {
    const next = await getMe()
    setViewer(client, next)
    return next
  }, [client])
  const query = useQuery({ queryKey: ['me'], queryFn: loadMe, retry: false })
  const me = query.data ?? guest

  const refresh = useCallback(async () => {
    try {
      await client.fetchQuery({ queryKey: ['me'], queryFn: loadMe })
    } catch {
      setViewer(client, guest)
    }
  }, [client, loadMe])

  const loginMutation = useMutation({
    mutationFn: (body: { name: string; password: string }) => apiData(api.POST('/api/auth/login', { body })),
    onSuccess: (next) => setViewer(client, next)
  })

  const registerMutation = useMutation({
    mutationFn: (body: { name: string; mail: string; password: string }) => apiData(api.POST('/api/auth/register', { body })),
    onSuccess: (next) => setViewer(client, next)
  })

  const logoutMutation = useMutation({
    mutationFn: () => apiEmpty(api.POST('/api/auth/logout')),
    onSuccess: () => setViewer(client, guest)
  })

  const login = useCallback(async (name: string, password: string) => {
    await loginMutation.mutateAsync({ name, password })
  }, [loginMutation])

  const register = useCallback(async (values: { name: string; mail: string; password: string }) => {
    await registerMutation.mutateAsync(values)
  }, [registerMutation])

  const logout = useCallback(async () => {
    await logoutMutation.mutateAsync()
  }, [logoutMutation])

  const setAvatar = useCallback((next: string) => {
    client.setQueryData<Me>(['me'], (current) => ({ ...(current ?? guest), avatar: next }))
  }, [client])

  const value = useMemo<SessionState>(() => {
    const signedIn = me.id > 0
    const role: Role = !signedIn ? 'guest' : me.admin ? 'admin' : 'user'
    return {
      role,
      admin: me.admin,
      signedIn,
      name: me.name,
      avatar: me.avatar,
      mustChangePassword: me.mustChangePassword,
      refresh,
      login,
      register,
      logout,
      setAvatar
    }
  }, [login, logout, me, refresh, register, setAvatar])

  return <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
}

export function useSession() {
  const value = useContext(SessionContext)
  if (!value) {
    throw new Error('SessionProvider is missing')
  }
  return value
}
