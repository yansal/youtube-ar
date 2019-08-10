import React from 'react'

import { API_URL } from '../constants/api'

class LogDetail extends React.Component {
  state = {
    logs: [],
  }

  componentDidMount() {
    this.refresh()
    this.refreshInterval = setInterval(this.refresh, 1000)
  }

  componentWillUnmount() {
    clearInterval(this.refreshInterval)
  }

  refresh = () => {
    const { match } = this.props
    const { id } = match && match.params
    const { nextCursor } = this.state

    fetch(`${API_URL}/urls/${id}/logs?cursor=${nextCursor || 0}`).then(response => {
      response.json().then(resource => {
        if (resource.logs === null) {
          return
        }
        const { logs } = this.state
        const updatedLogs = logs.concat(resource.logs)
        this.setState({ logs: updatedLogs, nextCursor: resource.next_cursor })
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
