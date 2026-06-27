const defaultPage = 1
const defaultPageSize = 20

export function pageFromParams(params: URLSearchParams) {
  return positiveInt(params.get('page'), defaultPage)
}

export function pageSizeFromParams(params: URLSearchParams) {
  return positiveInt(params.get('pageSize'), defaultPageSize)
}

export function setPageParams(params: URLSearchParams, page: number, pageSize: number) {
  const next = new URLSearchParams(params)
  if (page > defaultPage) {
    next.set('page', String(page))
  } else {
    next.delete('page')
  }
  if (pageSize !== defaultPageSize) {
    next.set('pageSize', String(pageSize))
  } else {
    next.delete('pageSize')
  }
  return next
}

function positiveInt(value: string | null, fallback: number) {
  if (!value) {
    return fallback
  }
  const parsed = Number(value)
  return Number.isInteger(parsed) && parsed > 0 ? parsed : fallback
}
