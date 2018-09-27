const s = new Set<string>()

export function has(value: string) {
	return s.has(value)
}

export function put(value: string, timeout: number = 60000) {
	s.add(value)
	if (timeout >= 0) {
		setTimeout(() => {
			s.delete(value)
		}, timeout)
	}
}

export function del(value: string) {
	return s.delete(value)
}
