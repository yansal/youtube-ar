import React from 'react'

import { API_URL } from '../constants/api'
import Video from './Video'

class VideoList extends React.Component {
  constructor(props) {
    super(props)

    this.videoInput = React.createRef()

    this.state = {
      list: []
    }
  }

  componentDidMount() {
    fetch(`${API_URL}/urls`).then(response => {
      response.json().then(resource => {
        this.setState({ list: resource.urls, nextCursor: resource.next_cursor })
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
        const updatedList = [res, ...list]

        this.setState({
          list: updatedList
        })
      })
    })
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
        this.setState({ list: updatedList, nextCursor: resource.next_cursor})
      })
    })
  }

  render() {
    const { list } = this.state
    const listNodes = list.map(video => (
      <Video key={video.id} video={video} />
    ))

    return (
      <div>
        <form onSubmit={this.handleSubmit}>
          <input type="text" ref={this.videoInput} />
        </form>
        <ul>{listNodes}</ul>
        <button onClick={this.handleNext}>Next</button>
      </div>
    )
  }
}

export default VideoList
