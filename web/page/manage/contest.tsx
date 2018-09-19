import * as React from 'react'

import { message, Button, Card, Col, Divider, Icon, Input, Modal, Popconfirm, Row, Table } from 'antd'
import { WrappedFormUtils } from 'antd/lib/form/Form'

import WrappedContestForm from '../../component/form/contest'
import { delContest, getContests, getProblems, hasToken, postContest, putContest, putProblem } from '../../model'
import { HistoryProps, IContest, IProblem } from '../../util/interface'
import { updateState } from '../../util/state'
import { renderType } from '../contest'

class ContestProblems extends React.Component<{ cid: string }> {
	public state = {
		loading: true,
		pid: '', pkey: '',
		problems: [] as IProblem[],
		pagination: {
			current: 1, pageSize: 50,
			total: 0, hideOnSinglePage: true
		}
	}
	private handleChange = (pagination = this.state.pagination) => {
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		this.setState({ loading: true, pagination: pager })
		const { pageSize: size, current: page } = pager
		getProblems({ page, size, all: 1, cid: this.props.cid })
			.then(({ total, list: problems }) => {
				this.state.pagination.total = total
				this.setState({
					pagination: this.state.pagination,
					loading: false, problems
				})
			})
			.catch((err) => {
				message.error(err)
				this.setState({ loading: false })
			})
	}
	private add = () => {
		const { pid: id, pkey: key } = this.state
		if (!id || !key) { return }
		putProblem(this.state.pid, { contest: { id, key } })
			.then(() => this.handleChange())
			.catch(message.error)
	}
	private del = (id: any) => {
		putProblem(id, { contest: null })
			.then(() => this.handleChange())
			.catch(message.error)
	}
	public componentWillMount() {
		updateState({ path: [ 'Manage', 'Contest' ] })
		this.handleChange()
	}
	public render() {
		return <React.Fragment>
			<Row gutter={16}>
				<Col span={10}>
					<Input
						placeholder="Problem ID"
						onChange={(e) => this.setState({ pid: e.target.value })}
						prefix={<Icon type="file-text" style={{ color: 'rgba(0,0,0,.25)' }} />}
					/>
				</Col>
				<Col span={10}>
					<Input
						placeholder="Problem Key (A, B, C, D ...)"
						onChange={(e) => this.setState({ pkey: e.target.value })}
						prefix={<Icon type="bars" style={{ color: 'rgba(0,0,0,.25)' }} />}
					/>
				</Col>
				<Col span={4}>
					<Button block type="primary" onClick={this.add}>Add</Button>
				</Col>
			</Row>
			<div className="divider" />
			<Table
				rowKey="_id"
				size="middle"
				loading={this.state.loading}
				dataSource={this.state.problems}
				pagination={this.state.pagination}
				onChange={this.handleChange}
				columns={[
					{ title: 'Problem ID', width: 250, dataIndex: '_id', render: (t) => (
						<a href={`/problem/${t}`} target="_blank">{t}</a>
					) },
					{ title: 'Problem Key', dataIndex: 'contest.key' },
					{ title: 'Problem Title', dataIndex: 'title' },
					{ title: 'Action', key: 'action', render: (t, r) => <Popconfirm
						title="Delete this problem?" onConfirm={() => this.del(r._id)}
					>
						<a style={{ color: 'red' }}>Delete</a>
					</Popconfirm> }
				]}
			/>
		</React.Fragment>
	}
}

export default class extends React.Component<HistoryProps> {
	private form: WrappedFormUtils = undefined
	public state = {
		loading: true,
		contests: [] as IContest[],
		pagination: { current: 1, pageSize: 50, total: 0 },
		modalTitle: '',
		modalOpen: false,
		modalContest: undefined as IContest
	}
	private handleChange = (pagination = this.state.pagination) => {
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		this.setState({ loading: true, pagination: pager })
		const { pageSize: size, current: page } = pager
		getContests({ page, size })
			.then(({ total, list: contests }) => {
				this.state.pagination.total = total
				this.setState({
					pagination: this.state.pagination,
					loading: false, contests
				})
			})
			.catch((err) => {
				message.error(err)
				this.setState({ loading: false })
			})
	}
	private openModal = (contest?: IContest) => {
		this.setState({
			modalOpen: true,
			modalTitle: contest ? 'Edit' : 'Create',
			modalContest: contest
		})
	}
	private ok = () => {
		this.form.validateFields((error, values) => {
			if (!error) {
				this.setState({ loading: true })
				if (this.state.modalContest) {
					putContest(this.state.modalContest._id, values)
						.then(() => {
							message.success('update success')
							this.setState({ modalOpen: false })
							this.handleChange()
						})
						.catch((err) => {
							message.error(err)
							this.setState({ loading: false })
						})
				} else {
					postContest(values)
						.then(() => {
							message.success('create success')
							this.setState({ modalOpen: false })
							this.handleChange()
						})
						.catch((err) => {
							message.error(err)
							this.setState({ loading: false })
						})
				}
			}
		})
	}
	private del = (id: any) => {
		delContest(id)
			.then(() => this.handleChange())
			.catch(message.error)
	}
	public componentWillMount() {
		updateState({ path: [ 'Manage', 'Contest' ] })
		if (hasToken()) { this.handleChange() }
	}
	public render() {
		return <React.Fragment>
			<Card
				title="Contests"
				extra={<React.Fragment>
					<Button onClick={() => this.openModal()}>
						Create Contest
					</Button>
				</React.Fragment>}
			>
				<Modal
					style={{ minWidth: 1000 }}
					destroyOnClose={true}
					visible={this.state.modalOpen}
					title={this.state.modalTitle}
					confirmLoading={this.state.loading}
					onOk={() => this.ok()}
					onCancel={() => this.setState({ modalOpen: false })}
				>
					<WrappedContestForm
						value={this.state.modalContest}
						wrappedComponentRef={(w: any) => {
							this.form = w && w.props.form
						}}
					/>
				</Modal>
				<Table
					rowKey="_id"
					size="middle"
					expandedRowRender={(t) => <ContestProblems cid={t._id} />}
					loading={this.state.loading}
					dataSource={this.state.contests}
					pagination={this.state.pagination}
					onChange={this.handleChange}
					columns={[
						{ title: 'ID', width: 250, dataIndex: '_id', render: (t) => (
							<a href={`/contest/${t}`} target="_blank">{t}</a>
						) },
						{ title: 'Title', dataIndex: 'title' },
						{ title: 'Type', dataIndex: 'type', render: renderType },
						{ title: 'Start At', dataIndex: 'startAt', render: (t) => new Date(t).toLocaleString() },
						{ title: 'End At', dataIndex: 'endAt', render: (t) => new Date(t).toLocaleString() },
						{ title: 'Action', key: 'action', render: (t, r) => <React.Fragment>
							<a onClick={() => this.openModal(r)}>Edit</a>
							<Divider type="vertical" />
							<Popconfirm title="Delete this contest?" onConfirm={() => this.del(r._id)}>
								<a style={{ color: 'red' }}>Delete</a>
							</Popconfirm>
						</React.Fragment> }
					]}
				/>
			</Card>
		</React.Fragment>
	}
}
