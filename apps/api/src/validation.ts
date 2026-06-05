import { z } from 'zod'

export const numericId = z.coerce.number().int().positive()
