import { IResult, Status } from './interface'

export interface Pack {
	_id: string
	result: IResult
	cases: IResult[]
}

export const SE = (id: string, e?: any): Pack => ({
	_id: id,
	result: {
		time: 0,
		memory: 0,
		status: Status.SE,
		extra: e && (e.message || e)
	},
	cases: []
})
