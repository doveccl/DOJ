import type { ReactNode } from 'react'

import { useLocale } from '../../../locale'

export type Option<T extends string | number> = { value: T; label: string }

export type Block = ReactNode

export function useAdminText() {
  return useLocale().text
}
