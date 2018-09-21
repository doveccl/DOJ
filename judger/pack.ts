import { IResult, Status } from '../common/interface'

interface Pack {
	_id: string
	result: IResult
	cases: IResult[]
}

export const SE = (id: string, err?: string): Pack => ({
	_id: id,
	result: {
		time: 0, memory: 0,
		status: Status.SE, extra: err
	},
	cases: []
})
