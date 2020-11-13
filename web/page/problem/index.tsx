import React from 'react'
import { withRouter } from 'react-router-dom'

import { message, Card, Input, Progress, Table, Tag } from 'antd'

import { LoginTip } from '../../component/login-tip'
import { getProblems, hasToken } from '../../model'
import { HistoryProps, IProblem } from '../../util/interface'
import { updateState } from '../../util/state'

class Problems extends React.Component<HistoryProps> {
	public state = {
		loading: true,
		search: '',
		problems: [] as IProblem[],
		pagination: { current: 1, pageSize: 50, total: 0 }
	}
	private handleChange = (pagination: any) => {
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		this.setState({ loading: true, pagination: pager })
		const { pageSize: size, current: page } = pager
		getProblems({ page, size, search: this.state.search })
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
	public componentDidMount() {
		updateState({ path: [ 'Problem' ] })
		if (hasToken()) { this.handleChange(this.state.pagination) }
	}
	public render() {
		return <React.Fragment>
			<LoginTip />
			<Card
				title="Problems"
				className="list"
				extra={<Input.Search
					placeholder="Title, content or tags"
					onSearch={this.onSearch}
				/>}
			>
				<Table
					rowKey="_id"
					size="middle"
					onRow={({ _id }) => ({
						onClick: () => this.props.history.push(`/problem/${_id}`)
					})}
					loading={this.state.loading}
					dataSource={this.state.problems}
					pagination={this.state.pagination}
					onChange={this.handleChange}
					columns={[
						{ title: 'Title', width: 100, dataIndex: 'title' },
						{ title: 'Tags', width: 200, key: 'tags', render: (t, r) => <React.Fragment>
							{r.tags.map((tag, i) => <Tag key={i}>{tag}</Tag>)}
						</React.Fragment> },
						{ title: 'Ratio', width: 100, key: 'ratio', render: (t, r) =>
							<Progress percent={Math.floor(100 * r.solve / r.submit)} status="active" />
						}
					]}
				/>
			</Card>
		</React.Fragment>
	}
}

export default withRouter(({ history }) => <Problems history={history} />)
