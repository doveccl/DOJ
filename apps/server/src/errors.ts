import type { Context } from 'hono'
import type { ContentfulStatusCode } from 'hono/utils/http-status'
import type { ZodIssue } from 'zod'

export interface ApiErrorIssue {
  path: string
  message: string
}

export class ApiHttpError extends Error {
  constructor(
    public status: ContentfulStatusCode,
    public code: string,
    message: string,
    public issues?: ApiErrorIssue[]
  ) {
    super(message)
  }
}

export function apiError(
  c: Context,
  status: ContentfulStatusCode,
  code: string,
  message: string,
  issues?: ApiErrorIssue[]
) {
  return c.json(
    {
      error: {
        code,
        message,
        ...(issues?.length ? { issues } : {})
      }
    },
    status
  )
}

export function notFound(c: Context, message = 'Resource not found') {
  return apiError(c, 404, 'NOT_FOUND', message)
}

export function validationIssues(issues: ZodIssue[]): ApiErrorIssue[] {
  return issues.map((issue) => ({
    path: issue.path.join('.'),
    message: issue.message
  }))
}
