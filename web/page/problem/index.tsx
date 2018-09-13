import * as React from 'react'
import { withRouter } from 'react-router-dom'
import { Card, Table, Input, Progress, Tag, message } from 'antd'

import * as model from '../../model'
import LoginTip from '../../component/login-tip'
import { IProblem, HistoryProps } from '../../util/interface'
import { updateState } from '../../util/state'

class Problems extends React.Component<HistoryProps> {
	state = {
		loading: true,
		search: '',
		problems: [] as IProblem[],
		pagination: { current: 1, pageSize: 50, total: 0 }
	}
	componentWillMount() {
		updateState({ path: [ 'Problem' ] })
		if (!model.hasToken()) { return }
		this.handleChange(this.state.pagination)
	}
	onSearch = (value: string) => {
		const pagination = { ...this.state.pagination }
		pagination.current = 1
		this.setState({ search: value })
		this.handleChange(pagination, value)
	}
	handleChange = (pagination: any, keyword?: any) => {
		const search = typeof keyword === 'string' ?
			keyword : this.state.search
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		this.setState({ loading: true, pagination: pager })
		const { pageSize: size, current: page } = pager
		model.getProblems({ page, size, search })
			.then(({ total, list: problems }) => {
				this.state.pagination.total = total
				this.setState({
					pagination: this.state.pagination,
					loading: false, problems
				})
			})
			.catch(err => {
				message.error(err)
				this.setState({ loading: false })
			})
	}
	render() {
		return <React.Fragment>
			<LoginTip />
			<Card
				title="Problems"
				className="problems"
				extra={<Input.Search
					placeholder="Title, content or tags"
					onSearch={this.onSearch}
				/>}
			>
				<Table
					rowKey="_id"
					onRow={({ _id }) => ({
						onClick: () => this.props.history.push(`/problem/${_id}`)
					})}
					loading={this.state.loading}
					dataSource={this.state.problems}
					pagination={this.state.pagination}
					onChange={this.handleChange}
					columns={[
						{ title: 'Title', width: 20, dataIndex: 'title' },
						{ title: 'Tags', width: 20, key: 'tags', render: (t, r) => <React.Fragment>
							{r.tags.map((tag, i) => <Tag key={i}>{tag}</Tag>)}
						</React.Fragment> },
						{ title: 'Ratio', width: 10, key: 'ratio', render: (t, r) =>
							<Progress percent={r.solve / r.submit} />
						}
					]}
				/>
			</Card>
		</React.Fragment>
	}
}

export default withRouter(({ history }) => <Problems history={history} />)
