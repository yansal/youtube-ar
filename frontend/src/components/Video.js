import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-router-dom'

import { BASE_URL } from '../constants/api'

class Video extends React.Component {
  constructor(props) {
    super(props)

    this.refreshInterval = setInterval(this.refresh, 1000)

    this.state = {
      status: props.video.status
    }
  }

  refresh = () => {
    const { id } = this.props && this.props.video
    const { status } = this.state

    if (status !== 'pending' && status !== 'processing') {
      return clearInterval(this.refreshInterval)
    }

    fetch(`${BASE_URL}/urls/${id}`).then(response => {
      response.json().then(video => {
        this.setState({
          status: video.status
        })
      })
    })
  }

  render() {
    const { status } = this.state
    const { video } = this.props

    return (
      <Link to={`/logs/${video.id}`}>
        <li>
          {video.url} ({status})
        </li>
      </Link>
    )
  }
}

Video.propTypes = {
  video: PropTypes.shape({
    id: PropTypes.number,
    status: PropTypes.string,
    url: PropTypes.string,
  })
}

export default Video
