export interface ILanguage {
	name: string
	suffix: string
}

export interface IConfig {
	_id: string
	value: any
}

export enum Group { common, admin, root }
export interface IUser<I, T> {
	_id: I
	name: string
	mail: string
	group: Group
	password: string
	solve: number
	submit: number
	introduction: string
	createdAt: T
	updatedAt: T
}

export interface IProblem<I, T> {
	_id: I
	title: string
	content: string
	tags: string[]
	timeLimit: number
	memoryLimit: number
	solve: number
	submit: number
	data?: I
	contest?: {
		id: I
		key: string
	}
	createdAt: T
	updatedAt: T
}

export enum ContestType { OI, ICPC }
export interface IContest<I, T> {
	_id: I
	title: string
	description: string
	type: ContestType
	startAt: T
	endAt: T
	freezeAt?: T
	createdAt: T
	updatedAt: T
}

export enum Status {
	WAIT, // Pending
	AC, // Accepted
	WA, // Wrong Answer
	TLE, // Time Limit Exceed
	MLE, // Memory Limit Exceed
	RE, // Runtime Error
	CE, // Compile Error
	SE, // System Error
	FREEZE // Freeze
}
export interface IResult {
	time: number
	memory: number
	status: Status
	extra?: string
}
export interface ISubmission<I, T> {
	_id: I
	uid: I
	pid: I
	cid?: I
	code: string
	language: number
	open: boolean
	result: IResult
	cases: IResult[]
	createdAt: T
	updatedAt: T
}

export interface IPost<I, T> {
	_id: I
	uid: I
	topic: I
	content: string
	createdAt: T
	updatedAt: T
}

export interface IFile<I, T> {
	_id: I
	filename: string
	contentType: string
	length: number
	chunkSize: number
	uploadDate: T
	metadata: any
	aliases: string[]
	md5: string
}
