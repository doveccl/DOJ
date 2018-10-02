import { Middleware } from 'koa'
import { Schema } from 'mongoose'

import { Contest, DContest } from '../model/contest'
import { DFile, File } from '../model/file'
import { DPost, Post } from '../model/post'
import { DProblem, Problem } from '../model/problem'
import { DSubmission, Submission } from '../model/submission'
import { DUser, User } from '../model/user'

type IDLike = string | Schema.Types.ObjectId

declare module 'koa' {
	interface Context {
		user?: DUser
		problem?: DProblem
		contest?: DContest
		submission?: DSubmission
		post?: DPost
		file?: DFile
	}
}

interface Cache<T> {
	[index: string]: T
}

const cache = {
	user: {} as Cache<DUser>,
	problem: {} as Cache<DProblem>,
	contest: {} as Cache<DContest>
}

export function clearCache() {
	cache.user = {}
	cache.problem = {}
	cache.contest = {}
}

export async function user(maybeID: IDLike) {
	const id = String(maybeID)
	if (cache.user[id]) { return cache.user[id] }
	return cache.user[id] = await User.findById(id)
}

export async function problem(maybeID: IDLike) {
	const id = String(maybeID)
	if (cache.problem[id]) { return cache.problem[id] }
	return cache.problem[id] = await Problem.findById(id)
}

export async function contest(maybeID: IDLike) {
	const id = String(maybeID)
	if (cache.contest[id]) { return cache.contest[id] }
	return cache.contest[id] = await Contest.findById(id)
}

export function fetch(
	type:
		'user' |
		'problem' |
		'contest' |
		'submission' |
		'post' |
		'file'
): Middleware {
	return async (ctx, next) => {
		const { id } = ctx.params
		if (id) {
			switch (type) {
				case 'user':
					ctx[type] = await user(id)
					break
				case 'problem':
					ctx[type] = await problem(id)
					break
				case 'contest':
					ctx[type] = await contest(id)
					break
				case 'submission':
					ctx[type] = await Submission.findById(id)
					break
				case 'post':
					ctx[type] = await Post.findById(id)
					break
				case 'file':
					ctx[type] = await File.findById(id)
			}
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
