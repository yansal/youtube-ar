import React from 'react'

import { API_URL } from '../constants/api'
import Video from './Video'

class VideoList extends React.Component {
  constructor(props) {
    super(props)

    this.videoInput = React.createRef()

    this.state = {
      list: [],
      filter: 'all'
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

  handleFilterChange = event => {
    this.setState({ filter: event.target.value }, this.refresh)
  }

  handleNext = event => {
    const { nextCursor } = this.state
    fetch(`${API_URL}/urls?cursor=${nextCursor}`).then(response => {
      response.json().then(resource => {
        if (resource.urls === null) {
          // TODO: remove next button?
          return
        }

        const { list } = this.state
        const updatedList = list.concat(resource.urls)
        this.setState({ list: updatedList, nextCursor: resource.next_cursor })
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
      response.json().then(res => {
        const { list } = this.state
        const updatedList = (list && [res, ...list]) || [res]

        this.setState({
          list: updatedList
        })
      })
    })
  }

  refresh = () => {
    const { filter } = this.state
    const baseURL = `${API_URL}/urls`
    let url = filter === 'all' ? baseURL : `${baseURL}?status=${filter}`
    fetch(url).then(response => {
      response.json().then(resource => {
        this.setState({ list: resource.urls, nextCursor: resource.next_cursor })
      })
    })
  }

  render() {
    const { list } = this.state

    const listNodes = list && list.map(video => <Video key={video.id} video={video} onDelete={this.handleDelete} />)

    return (
      <div>
        <form onSubmit={this.handleSubmit}>
          <input type="text" ref={this.videoInput} />
        </form>
        <select value={this.state.filter} onChange={this.handleFilterChange}>
          <option value="all">All</option>
          <option value="success">Success</option>
          <option value="failure">Failure</option>
          <option value="processing">Processing</option>
          <option value="pending">Pending</option>
        </select>
        {list ? <ul className="yar-video-list">{listNodes}</ul> : <div>Nothing to show!</div>}
        <button onClick={this.handleNext}>Next</button>
      </div>
    )
  }
}

export default VideoList
