import { IResult, Status } from './interface'

export interface Pack {
	_id: string
	result: IResult
	cases: IResult[]
}

export const Case = (s: Status, t: number, m: number, e?: string): IResult => ({
	time: 1000 * t, memory: m, status: s, extra: e
})

export const CE = (id: string, e?: any): Pack => ({
	_id: id,
	cases: [],
	result: {
		time: 0,
		memory: 0,
		status: Status.CE,
		extra: e && (e.message || e)
	}
})

export const SE = (id: string, e?: any): Pack => ({
	_id: id,
	cases: [],
	result: {
		time: 0,
		memory: 0,
		status: Status.SE,
		extra: e && (e.message || e)
	}
})
