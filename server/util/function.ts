/**
 * Convert two values to string and compare
 * @param a the first value
 * @param b the second value
 */
export function toStringCompare<T>(a: T, b: T) {
	return String(a) === String(b)
}

/**
 * Validate password format
 * @param password the password
 * @param min min length
 * @param max max length
 */
export function validatePassword(password: string, min = 6, max = 20) {
	if (!password) {
		throw new Error('password is required')
	}
	if (password.length < min || max < password.length) {
		throw new Error('invalid password (length must be 6-20)')
	}
}
