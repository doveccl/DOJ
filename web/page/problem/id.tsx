import { useContext, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { message, Card, Divider, Tag } from 'antd'

import { parseMemory, parseTime } from '../../../common/function'
import { Discuss } from '../../component/discuss'
import { SubmitForm } from '../../component/form/submit'
import { LoginTip } from '../../component/login-tip'
import { MarkDown } from '../../component/markdown'
import { getProblem } from '../../model'
import { IProblem } from '../../interface'
import { GlobalContext } from '../../global'

export default function Problem() {
  const { id } = useParams()
  const [tab, setTab] = useState(location.hash || '#submit')
  const [problem, setProblem] = useState(null as IProblem)
  const [global, setGlobal] = useContext(GlobalContext)

  useEffect(() => setGlobal({
    path: [
      { url: '/problem', desc: 'Problem' },
      problem?.title ?? id
    ]
  }), [problem?.title])

  useEffect(() => {
    global.user && getProblem(id)
      .then(setProblem)
      .catch(message.error)
  }, [global.user])

  return <>
    <LoginTip />
    <Card
      loading={!problem}
      title={problem?.title ?? 'Problem'}
      extra={problem && <>
        <Link to={`/submission?pid=${problem._id}`}>
          Solutions ({problem.solve}/{problem.submit})
        </Link>
        <Divider type="vertical" />
        <Tag color="volcano">{parseTime(problem.timeLimit)}</Tag>
        <Tag color="orange">{parseMemory(problem.memoryLimit)}</Tag>
      </>}
    >
      <MarkDown trusted children={problem?.content} />
    </Card>
    <div className="divider" />
    <Card
      activeTabKey={tab}
      onTabChange={t => setTab(location.hash = t)}
      tabList={[
        { key: '#submit', tab: 'Submit' },
        { key: '#discuss', tab: 'Discuss' }
      ]}
    >
      {tab === '#discuss' ? <Discuss topic={id} /> : <SubmitForm pid={id} />}
    </Card>
  </>
}
