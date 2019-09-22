import * as React from 'react'

import { message, Button, Card, Divider, Input, Modal, Popconfirm, Table, Tag } from 'antd'
import { WrappedFormUtils } from 'antd/lib/form/Form'

import { parseMemory, parseTime } from '../../../common/function'
import { WrappedProblemForm } from '../../component/form/problem'
import { delProblem, getProblems, hasToken, postProblem, putProblem, rejudgeSubmission } from '../../model'
import { HistoryProps, IProblem } from '../../util/interface'
import { updateState } from '../../util/state'

export default class extends React.Component<HistoryProps> {
	private form: WrappedFormUtils = undefined
	public state = {
		loading: true,
		search: '',
		problems: [] as IProblem[],
		pagination: { current: 1, pageSize: 50, total: 0 },
		modalTitle: '',
		modalOpen: false,
		modalProblem: undefined as IProblem
	}
	private handleChange = (pagination = this.state.pagination) => {
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		this.setState({ loading: true, pagination: pager })
		const { pageSize: size, current: page } = pager
		getProblems({ page, size, all: 1, search: this.state.search })
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
	private onSearch = (value: string) => {
		const pagination = { ...this.state.pagination }
		pagination.current = 1
		this.setState({ search: value }, () => {
			this.handleChange(pagination)
		})
	}
	private openModal = (problem?: IProblem) => {
		this.setState({
			modalOpen: true,
			modalTitle: problem ? 'Edit' : 'Create',
			modalProblem: problem
		})
	}
	private ok = () => {
		this.form.validateFields((error: any, values: any) => {
			if (!error) {
				this.setState({ loading: true })
				if (this.state.modalProblem) {
					putProblem(this.state.modalProblem._id, values)
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
					postProblem(values)
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
	private rejudge = (pid: any) => {
		rejudgeSubmission({ pid })
			.then(() => message.success('rejudge success'))
			.catch(message.error)
	}
	private del = (id: any) => {
		delProblem(id)
			.then(() => this.handleChange())
			.catch(message.error)
	}
	public componentDidMount() {
		updateState({ path: [ 'Manage', 'Problem' ] })
		if (hasToken()) { this.handleChange() }
	}
	public render() {
		return <React.Fragment>
			<Card
				title="Problems"
				extra={<React.Fragment>
					<Input.Search
						style={{ width: 180 }}
						placeholder="Title, content or tags"
						onSearch={this.onSearch}
					/>
					<Divider type="vertical" />
					<Button onClick={() => this.openModal()}>
						Create Problem
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
					<WrappedProblemForm
						value={this.state.modalProblem}
						wrappedComponentRef={(w: any) => {
							this.form = w && w.props.form
						}}
					/>
				</Modal>
				<Table
					rowKey="_id"
					size="middle"
					loading={this.state.loading}
					dataSource={this.state.problems}
					pagination={this.state.pagination}
					onChange={this.handleChange}
					columns={[
						{ title: 'ID', width: 250, dataIndex: '_id', render: (t) => (
							<a href={`/problem/${t}`} target="_blank">{t}</a>
						) },
						{ title: 'Title', dataIndex: 'title' },
						{ title: 'Time', dataIndex: 'timeLimit', render: parseTime },
						{ title: 'Memory', dataIndex: 'memoryLimit', render: parseMemory },
						{ title: 'Tags', key: 'tags', render: (t, r) => <React.Fragment>
							{r.tags.map((tag, i) => <Tag key={i}>{tag}</Tag>)}
						</React.Fragment> },
						{ title: 'Data', width: 250, dataIndex: 'data', render: (t) => (
							t ? <a href={`/api/file/${t}`} target="_blank">{t}</a> : 'null'
						) },
						{ title: 'Action', key: 'action', render: (t, r) => <React.Fragment>
							<a onClick={() => this.openModal(r)}>Edit</a>
							<Divider type="vertical" />
							<Popconfirm title="Rejudge this problem?" onConfirm={() => this.rejudge(r._id)}>
								<a>Rejudge</a>
							</Popconfirm>
							<Divider type="vertical" />
							<Popconfirm title="Delete this problem?" onConfirm={() => this.del(r._id)}>
								<a style={{ color: 'red' }}>Delete</a>
							</Popconfirm>
						</React.Fragment> }
					]}
				/>
			</Card>
		</React.Fragment>
	}
}
