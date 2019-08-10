import React from 'react'
// import PropTypes from 'prop-types'

import { BASE_URL } from '../constants/api'

class LogDetail extends React.Component {
  state = {
    logs: []
  }

  componentDidMount() {
    const { match } = this.props
    const { id } = match && match.params

    fetch(`${BASE_URL}/urls/${id}/logs`).then(response => {
      response.json().then(logs => {
        this.setState({ logs })
      })
    })
  }

  render() {
    const { match } = this.props
    const { id } = match && match.params

    const { logs } = this.state
    const logNodes = logs.map(log => <li>{log.log}</li>)

    return <div>
      <h1>Logs for video #{id}</h1>
      <ul>
        {logNodes}
      </ul>
    </div>
  }
}

LogDetail.propTypes = {}

export default LogDetail
