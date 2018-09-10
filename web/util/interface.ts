import { History } from 'history'

export enum UserGroup { common, admin, root }

export interface HistoryProps {
	history: History
}

export interface IUser {
	_id: string
	name: string
	mail: string
	group: UserGroup
	password: string
	solve: number
	submit: number
	introduction: string
	createdAt: Date
	updatedAt: Date
}

export interface IProblem {
	_id: string
	title: string
	content: string
	tags: string[]
	timeLimit: number
	memoryLimit: number
	solve: number
	submit: number
	data?: string
	contest?: {
		id: string
		key: string
	}
	createdAt: Date
	updatedAt: Date
}

export enum ContestType { OI, ICPC }

export interface IContest {
	_id: string
	title: string
	description: string
	type: ContestType
	startAt: Date
	endAt: Date
	freezeAt?: Date
	createdAt: Date
	updatedAt: Date
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
	OTHER // Others
}

export interface IResult {
	time: number
	memory: number
	status: Status
}

export interface ISubmission {
	_id: string
	uid: string
	pid: string
	cid?: string
	code: string
	language: number
	open: boolean
	result: IResult
	cases: IResult[]
	createdAt: Date
	updatedAt: Date
}

export interface IFile {
	_id: string
	filename: string
	contentType: string
	length: number
	chunkSize: number
	uploadDate: Date
	metadata: any
	aliases: string[]
	md5: string
}
