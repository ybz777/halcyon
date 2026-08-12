export interface PaginatedResponse<T> {
  items: T[]
  next_cursor?: string
  has_more: boolean
}

export interface ProblemDetails {
  type?: string
  title?: string
  status?: number
  detail?: string
  instance?: string
  errors?: FieldError[]
}

export interface FieldError {
  field: string
  code?: string
  message?: string
}
