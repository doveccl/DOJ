import md5 from 'md5'

/**
 * Convert two values to string and compare
 * @param a the first value
 * @param b the second value
 */
export function compare<T>(a: T, b: T) {
	return String(a) === String(b)
}

/**
 * Check password format
 * @param pass the password
 * @param err throw error
 * @param min min length
 * @param max max length
 */
export function checkPassword(pass: string, err = false, min = 6, max = 20) {
	if (!err) { return pass && min <= pass.length && pass.length <= max }
	if (!checkPassword(pass, false, min, max)) {
		throw new Error(`length of password should be ${min}-${max}`)
	}
}

/**
 * Convert any value to integer
 * @param x any value
 */
export const Int = (x: any) => parseInt(x, 10)

/**
 * Get gravatar link by mail
 * @param m mail address
 * @param d gravatar param d
 */
export const glink = (m = '', d = 'wavatar') => (
	`//cdn.v2ex.com/gravatar/${md5(m)}?d=${d}`
)

/**
 * Parse time usage
 * @param t time in ms
 */
export const parseTime = (t: number) => `${Math.round(1000 * t)} ms`

/**
 * Parse memory usage
 * @param b memory in bytes
 */
export const parseMemory = (b: number) => {
	const k = b / 1024
	const m = k / 1024
	if (b < 1024) { return `${b} B` }
	if (k < 1024) { return `${k.toFixed(1)} KiB` }
	if (m < 1024) { return `${m.toFixed(1)} MiB` }
	return `${(m / 1024).toFixed(2)} GiB`
}

/**
 * Parse time diff count
 * @param t time in unix
 */
export function parseCount(t: number) {
	let tmp = Math.abs(Int(t / 1000))
	const sign = t < 0 ? '-' : '+'
	const s = ('0' + tmp % 60).slice(-2)
	tmp = Int(tmp / 60)
	const m = ('0' + tmp % 60).slice(-2)
	tmp = Int(tmp / 60)
	return `${sign} ${tmp}:${m}:${s}`
}
