import { IResult, Status } from '../common/interface'

interface Pack {
	result: IResult
	cases: IResult[]
}

export const SE = (err?: string): Pack => ({
	result: {
		time: 0, memory: 0,
		status: Status.SE, extra: err
	},
	cases: []
})
