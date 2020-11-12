import { History } from 'history'
import { match } from 'react-router-dom'
import { Moment } from 'moment'

import * as I from '../../common/interface'

export type ILanguage = Pick<I.ILanguage, 'name' | 'suffix'>

export type IUser = I.IUser<string, string>
export type IProblem = I.IProblem<string, string>
export type IContest = I.IContest<string, string>
export type IFile = I.IFile<string, string>

export interface ISubmission extends I.ISubmission<string, string> {
	uname: string
	ptitle: string
}

export interface IPost extends I.IPost<string, string> {
	uname: string
	umail: string
}

export interface HistoryProps {
	history: History
}

export interface MatchProps {
	match: match<any>
}
