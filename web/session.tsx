import { createContext, useCallback, useContext, useMemo } from 'react'
import type { ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { getMe, login as loginRequest, logout as logoutRequest, register as registerRequest } from './client'
import type { Me } from './client'

export type Role = 'guest' | 'user' | 'admin'

type SessionState = {
  role: Role
  admin: boolean
  signedIn: boolean
  name: string
  avatar?: string
  refresh: () => Promise<void>
  login: (name: string, password: string) => Promise<void>
  register: (values: { name: string; mail: string; password: string }) => Promise<void>
  logout: () => Promise<void>
  setAvatar: (avatar: string) => void
}

const SessionContext = createContext<SessionState | null>(null)

const guest: Me = {
  id: 0,
  name: '',
  mail: '',
  bio: '',
  avatar: '',
  admin: false
}

export function SessionProvider({ children }: { children: ReactNode }) {
  const client = useQueryClient()
  const query = useQuery({ queryKey: ['me'], queryFn: getMe, retry: false })
  const me = query.data ?? guest

  const refresh = useCallback(async () => {
    try {
      const next = await client.fetchQuery({ queryKey: ['me'], queryFn: getMe })
      client.setQueryData(['me'], next)
    } catch {
      client.setQueryData(['me'], guest)
    }
  }, [client])

  const loginMutation = useMutation({
    mutationFn: loginRequest,
    onSuccess: (next) => client.setQueryData(['me'], next)
  })

  const registerMutation = useMutation({
    mutationFn: registerRequest,
    onSuccess: (next) => client.setQueryData(['me'], next)
  })

  const logoutMutation = useMutation({
    mutationFn: logoutRequest,
    onSettled: () => client.setQueryData(['me'], guest)
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
