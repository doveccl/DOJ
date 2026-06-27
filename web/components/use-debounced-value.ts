import { useEffect, useState } from 'react'

export function useDebouncedValue<T>(value: T, delay = 250) {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const timer = window.setTimeout(() => setDebounced(value), delay)
    return () => window.clearTimeout(timer)
  }, [value, delay])

  return debounced
}

export function useRemoteSearch(delay = 250) {
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const searchText = useDebouncedValue(search.trim(), delay)

  return {
    active: open || searchText.length > 0,
    searchText,
    setOpen,
    setSearch
  }
}
