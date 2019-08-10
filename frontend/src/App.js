import React from 'react'
import { BrowserRouter, Route, Switch } from 'react-router-dom'

import LogDetail from './components/LogDetail'
import VideoList from './components/VideoList'

function App() {
  return (
    <BrowserRouter>
      <Switch>
        <Route exact path="/" component={VideoList}/>
        <Route path="/logs/:id" component={LogDetail} />
      </Switch>
    </BrowserRouter>
  )
}

export default App
