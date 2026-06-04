export interface PageQuery {
  page?: number
  size?: number
}

export interface PageResult<T> {
  total: number
  list: T[]
}
