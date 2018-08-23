/**
 * Convert two values to string and compare
 * @param a the first value
 * @param b the second value
 */
export function toStringCompare<T>(a: T, b: T) {
	return String(a) === String(b)
}
