import { useContext } from 'react'
import { Link } from 'react-router-dom'
import { Alert } from 'antd'
import { GlobalContext } from '../../global'

export function LoginTip() {
  const [global] = useContext(GlobalContext)
  return global.user ? null : <Alert
    type="warning"
    style={{
      margin: '-10px 0 10px 0'
    }}
    message={<span>
      This page is not available for guest,
      please {<Link to="/login">login</Link>} first
    </span>}
  />
}
