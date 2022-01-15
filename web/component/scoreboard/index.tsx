import React from 'react'

import { message, Table } from 'antd'

import { parseCount } from '../../../common/function'
import { ContestType, Status } from '../../../common/interface'
import { getContest, getProblems, getSubmissions } from '../../model'
import { IContest, IProblem, ISubmission } from '../../interface'

interface ScoreboardProps {
	id: string
}

const score = (s: ISubmission) => {
	let cnt = 0
	if (!s.cases) { return NaN }
	for (const c of s.cases) {
		if (c.status === Status.AC) { cnt++ }
	}
	return Math.round(100 * cnt / s.cases.length)
}

export class Scoreboard extends React.Component<ScoreboardProps> {
	private timer: any
	public state = {
		firstLoad: false,
		contest: {} as IContest,
		problems: [] as IProblem[],
		submissions: [] as ISubmission[]
	}
	private refresh = () => {
		if (new Date() > new Date(this.state.contest.endAt)) {
			clearInterval(this.timer)
		} else if (this.state.contest.type === ContestType.OI) {
			return 'result for OI is unavailable before contest end'
		}
		getSubmissions({ size: -1, cid: this.props.id })
			.then(({ list }) => this.setState({ submissions: list, firstLoad: true }))
			.catch(message.error)
	}
	private sort = () => {
		const result: any[] = []
		const um = new Map<string, number>()
		const { contest, problems, submissions } = this.state
		const ss = submissions.sort((a, b) => (
			a.createdAt.localeCompare(b.createdAt)
		))
		for (const s of ss) {
			if (s.result.status === Status.WAIT) {
				continue // ignore pending status
			}
			if (!um.has(s.uid)) {
				um.set(s.uid, result.length)
				result.push({ uname: s.uname })
			}
			const idx = um.get(s.uid)
			if (contest.type === ContestType.OI) {
				result[idx][s.pid] = score(s)
			} else {
				if (!result[idx].solve) {
					result[idx].solve = 0
				}
				if (!result[idx][s.pid]) {
					result[idx][s.pid] = {
						penalty: 0, fail: 0
					}
				}
				if (s.result.status === Status.AC) {
					if (!result[idx][s.pid].ac) {
						result[idx].solve++
						result[idx][s.pid].ac = parseCount(
							+new Date(s.createdAt) -
							+new Date(contest.startAt)
						)
					}
				} else if (s.result.status === Status.FREEZE) {
					if (!result[idx][s.pid].ac) {
						result[idx][s.pid].freeze = true
					}
				} else {
					result[idx][s.pid].fail--
					result[idx][s.pid].penalty += Math.floor((
						+new Date(s.createdAt) -
						+new Date(contest.startAt)
					) / 1000 / 60)
				}
			}
		}
		if (contest.type === ContestType.OI) {
			for (const u of result) {
				u.score = 0
				u.solve = 0
				for (const p of problems) {
					if (typeof u[p._id] === 'number') {
						u.score += u[p._id]
						if (u[p._id] === 100) {
							u.solve++
						}
					} else {
						u[p._id] = '-'
					}
				}
			}
			return result.sort((a, b) => b.score - a.score)
		}
		for (const u of result) {
			u.penalty = 0
			for (const p of problems) {
				if (!u[p._id]) {
					u[p._id] = '-'
				} else if (!u[p._id].ac) {
					u[p._id] = `(${u[p._id].fail})`
				} else {
					const fail = u[p._id].fail
					u.penalty += u[p._id].penalty
					u[p._id] = u[p._id].ac
					if (fail) { u[p._id] += `(${fail})` }
				}
			}
		}
		return result.sort((a, b) => (
			a.solve === b.solve ?
				a.penalty - b.penalty :
				b.solve - a.solve
		))
	}
	public componentDidMount() {
		this.refresh()
		getContest(this.props.id)
			.then((contest: IContest) => {
				this.setState({ contest })
				if (new Date() <= new Date(contest.endAt)) {
					this.timer = setInterval(this.refresh, 10000)
				}
			})
			.catch(message.error)
		getProblems({ cid: this.props.id })
			.then(({ list }) => this.setState({ problems: list }))
			.catch(message.error)
	}
	public componentWillUnmount() {
		clearInterval(this.timer)
	}
	public render() {
		const { contest, problems, firstLoad } = this.state
		const ps = problems.sort((a, b) => (
			a.contest.key.localeCompare(b.contest.key)
		))
		if (!contest._id || problems.length === 0 || !firstLoad) {
			return <Table loading={true} />
		}
		const oi = contest.type === ContestType.OI
		return <Table
			bordered
			size="middle"
			rowKey="uname"
			pagination={false}
			dataSource={this.sort()}
			columns={[
				{ title: 'User', dataIndex: 'uname' },
				{ title: 'Solve', dataIndex: 'solve' },
				{ title: oi ? 'Score' : 'Penalty', dataIndex: oi ? 'score' : 'penalty' }
			].concat(ps.map((p) => (
				{ title: p.contest.key, dataIndex: p._id }
			)))}
		/>
	}
}
