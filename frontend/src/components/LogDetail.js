import React from 'react'

import { API_URL } from '../constants/api'

class LogDetail extends React.Component {
  state = {
    logs: []
  }

  componentDidMount() {
    const { match } = this.props
    const { id } = match && match.params

    fetch(`${API_URL}/urls/${id}/logs`).then(response => {
      response.json().then(logs => {
        this.setState({ logs })
      })
    })
  }

  render() {
    const { match } = this.props
    const { id } = match && match.params

    const { logs } = this.state
    const logNodes = logs.map((log, index) => <li key={index}>{log.log}</li>)

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
