export interface PageQuery {
  page?: number
  pageSize?: number
}

export interface PageResult<T> {
  items: T[]
  page: number
  pageSize: number
  total: number
}
