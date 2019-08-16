import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-router-dom'

import { API_URL } from '../constants/api'

class Video extends React.Component {
  constructor(props) {
    super(props)

    this.state = {
      video: props.video
    }
  }

  componentDidMount() {
    this.refreshInterval = setInterval(this.refresh, 1000)
  }

  componentWillUnmount() {
    clearInterval(this.refreshInterval)
  }

  handleDelete = event => {
    this.props.onDelete(this.props.video.id)
  }

  handleRetry = event => {
    this.props.onRetry(this.props.video.id)
  }

  refresh = () => {
    const { video } = this.state

    if (video.status !== 'pending' && video.status !== 'processing') {
      return clearInterval(this.refreshInterval)
    }

    fetch(`${API_URL}/urls/${video.id}`).then(response => {
      response.json().then(video => {
        this.setState({ video })
      })
    })
  }

  getStatus = () => {
    const { video } = this.state
    switch (video.status) {
      case 'success':
        return null
      case 'failure':
        return video.error
      default:
        return video.status
    }
  }

  render() {
    const { video } = this.state

    return (
      <li className="yar-video-card">
        <a className="yar-video-card__image" href={video.url}>
          {video.oembed ? (
            <img alt={video.oembed.title} className="yar-video-card__thumbnail" src={video.oembed.thumbnail_url} />
          ) : (
            <img alt="placeholder" className="yar-video-card__thumbnail" src="https://via.placeholder.com/480x360" />
          )}
        </a>

        <div className="yar-video-card__title">
          <a href={video.url}>{video.oembed ? video.oembed.title : video.url}</a>
        </div>

        <div className="yar-video-card__actions">
          <div>{this.getStatus()}</div>

          <button onClick={this.handleDelete}>Delete</button>
          {video.status !== 'success' && <button onClick={this.handleRetry}>Retry</button>}
          {video.file && (
            <div>
              <a href={video.file} rel="noopener noreferrer" target="_blank">
                Download
              </a>
            </div>
          )}

          <Link to={`/logs/${video.id}`}>Show logs</Link>
        </div>
      </li>
    )
  }
}

Video.propTypes = {
  video: PropTypes.shape({
    id: PropTypes.number,
    status: PropTypes.string,
    url: PropTypes.string
  })
}

export default Video
