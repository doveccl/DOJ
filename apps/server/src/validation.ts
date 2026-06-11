import { z } from 'zod'

export const numericId = z.coerce.number().int().positive()

export const dateString = z.string().refine((value) => !Number.isNaN(Date.parse(value)), {
  message: 'Expected a valid date string'
})

// Shared count + offset pagination query for list endpoints.
export const listQuerySchema = z.object({
  page: z.coerce.number().int().positive().default(1),
  pageSize: z.coerce.number().int().min(1).max(100).default(50)
})

export function pageOffset(page: number, pageSize: number) {
  return (page - 1) * pageSize
}
