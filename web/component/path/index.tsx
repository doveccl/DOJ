import './index.less'
import { useContext } from 'react'
import { Link } from 'react-router-dom'
import { Breadcrumb } from 'antd'
import { GlobalContext } from '../../global'

export function Path() {
  const [global] = useContext(GlobalContext)
  return <Breadcrumb className="breadcrumb"
    items={[{ title: 'DOJ' }, ...(global.path ?? []).map(i => ({
      title: typeof i === 'string' ? i : <Link to={i.url}>{i.desc}</Link>
    }))]}
  />
}
