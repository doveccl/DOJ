import { prepareData } from './data'

export const judge = async (s: any) => {
	const { language, code, data, timeLimit, memoryLimit } = s
	await prepareData(data)
}
