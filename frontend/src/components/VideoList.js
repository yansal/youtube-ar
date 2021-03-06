import React from 'react'

import { API_URL } from '../constants/api'
import Video from './Video'

class VideoList extends React.Component {
  constructor(props) {
    super(props)

    this.videoInput = React.createRef()

    this.state = {
      list: [],
      search: '',
      status: 'all'
    }
  }

  componentDidMount() {
    this.refresh()
  }

  handleDelete = id => {
    fetch(`${API_URL}/urls/${id}`, {
      method: 'DELETE'
    }).then(response => {
      const { list } = this.state
      const updatedList = list.filter(video => video.id !== id)

      this.setState({
        list: updatedList
      })
    })
  }

  handleStatusChange = event => {
    this.setState({ status: event.target.value, nextCursor: 0 }, this.refresh)
  }

  handleSearchChange = event => {
    this.setState({ search: event.target.value, nextCursor: 0 }, this.refresh)
  }

  handleNext = event => {
    fetch(this.buildURL()).then(response => {
      response.json().then(resource => {
        if (resource.urls === null) {
          return
        }

        const { list } = this.state
        const updatedList = list.concat(resource.urls)
        this.setState({ list: updatedList, nextCursor: resource.next_cursor })
      })
    })
  }

  handleRetry = id => {
    fetch(`${API_URL}/urls/${id}/retry`, {
      method: 'POST'
    }).then(response => {
      response.json().then(resource => {
        this.addVideo(resource)
      })
    })
  }

  handleSubmit = event => {
    event.preventDefault()

    const { value } = this.videoInput && this.videoInput.current

    fetch(`${API_URL}/urls`, {
      method: 'POST',
      body: JSON.stringify({
        url: value
      })
    }).then(response => {
      response.json().then(resource => {
        this.addVideo(resource)
      })
    })
  }

  addVideo = resource => {
    const { list } = this.state
    const updatedList = (list && [resource, ...list]) || [resource]

    this.setState({
      list: updatedList
    })
  }

  refresh = () => {
    fetch(this.buildURL()).then(response => {
      response.json().then(resource => {
        this.setState({ list: resource.urls, nextCursor: resource.next_cursor })
      })
    })
  }

  buildURL = () => {
    const { search, status, nextCursor } = this.state
    var params = new URLSearchParams()
    if (search && search !== '') {
      params.set('q', search)
    }
    if (status && status !== 'all') {
      params.set('status', status)
    }
    if (nextCursor && nextCursor !== 0) {
      params.set('cursor', nextCursor)
    }
    return `${API_URL}/urls?${params.toString()}`
  }

  render() {
    const { nextCursor, list } = this.state

    const listNodes =
      list &&
      list.map(video => <Video key={video.id} video={video} onDelete={this.handleDelete} onRetry={this.handleRetry} />)

    return (
      <div>
        <form onSubmit={this.handleSubmit}>
          <input type="text" ref={this.videoInput} placeholder="Paste a URL" />
        </form>
        <select value={this.state.status} onChange={this.handleStatusChange}>
          <option value="all">All</option>
          <option value="success">Success</option>
          <option value="failure">Failure</option>
          <option value="processing">Processing</option>
          <option value="pending">Pending</option>
        </select>
        <input type="text" onChange={this.handleSearchChange} placeholder="Search" />
        {list ? <ul className="yar-video-list">{listNodes}</ul> : <div>Nothing to show!</div>}
        {nextCursor > 0 && <button onClick={this.handleNext}>Next</button>}
      </div>
    )
  }
}

export default VideoList
