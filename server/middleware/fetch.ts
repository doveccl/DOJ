import { Middleware } from 'koa'

import { Contest, IContest } from '../model/contest'
import { File, IFile } from '../model/file'
import { IPost, Post } from '../model/post'
import { IProblem, Problem } from '../model/problem'
import { ISubmission, Submission } from '../model/submission'
import { IUser, User } from '../model/user'

declare module 'koa' {
	interface Context {
		user?: IUser
		problem?: IProblem
		contest?: IContest
		submission?: ISubmission
		post?: IPost
		file?: IFile
	}
}

interface Cache<T> {
	[index: string]: T
}

const cache = {
	user: {} as Cache<IUser>,
	problem: {} as Cache<IProblem>,
	contest: {} as Cache<IContest>
}

export async function user(id: any) {
	if (cache.user[id]) { return cache.user[id] }
	return cache.user[id] = await User.findById(id)
}

export async function problem(id: any) {
	if (cache.problem[id]) { return cache.problem[id] }
	return cache.problem[id] = await Problem.findById(id)
}

export async function contest(id: any) {
	if (cache.contest[id]) { return cache.contest[id] }
	return cache.contest[id] = await Contest.findById(id)
}

export async function fetch(
	type:
		'user' |
		'problem' |
		'contest' |
		'submission' |
		'post' |
		'file',
	id: any
) {
	switch (type) {
		case 'user': return await user(id)
		case 'problem': return await problem(id)
		case 'contest': return await contest(id)
		case 'submission': return await Submission.findById(id)
		case 'post': return await Post.findById(id)
		case 'file': return await File.findById(id)
	}
}

export function urlFetch(
	type:
		'user' |
		'problem' |
		'contest' |
		'submission' |
		'post' |
		'file'
): Middleware {
	return async (ctx, next) => {
		if (ctx.params.id) {
			ctx[type] = await fetch(type, ctx.params.id)
			if (!ctx[type]) { throw new Error(`${type} not found`) }
		}
		await next()
		/**
		 * delete resource cache
		 * if it was removed from database
		 */
		if (ctx.params.id && ctx.method === 'DELETE') {
			switch (type) {
				case 'file':
				case 'post':
				case 'submission': break
				default: delete cache[type][ctx.params.id]
			}
		}
	}
}
