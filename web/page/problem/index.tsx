import * as React from 'react'
import { Card, Table, Input, Progress, Tag, message, Pagination } from 'antd'

import * as model from '../../model'
import LoginTip from '../../component/login-tip'
import { IProblem } from '../../util/interface'
import { updateState, globalState } from '../../util/state'

export default class extends React.Component {
	state = {
		loading: true,
		problems: [] as IProblem[],
		pagination: { current: 1, pageSize: 50, total: 0 }
	}
	componentWillMount() {
		updateState({ path: [ 'Problem' ] })
		this.handleChange(this.state.pagination)
	}
	handleChange = (pagination: any, search?: any) => {
		const pager = { ...this.state.pagination }
		pager.current = pagination.current
		this.setState({ loading: true, pagination: pager })
		const { pageSize: size, current: page } = pagination
		model.getProblems({ page, size, search })
			.then(data => {
				this.state.pagination.total = data.total
				this.setState({
					pagination: this.state.pagination,
					problems: data.list,
					loading: false
				})
			})
			.catch(err => globalState.user && message.error(err))
	}
	render() {
		return <React.Fragment>
			<LoginTip />
			<Card
				title="Problems"
				extra={<Input.Search
					placeholder="Title, content or tags"
					onSearch={value => this.handleChange(this.state.pagination, value)}
				/>}
			>
				<Table
					rowKey="_id"
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
