import { z } from 'zod'

export const numericId = z.coerce.number().int().positive()

export const dateString = z.string().refine((value) => !Number.isNaN(Date.parse(value)), {
  message: 'Expected a valid date string'
})
