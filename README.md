# Reactor

[![Build Status](https://secure.travis-ci.org/jcoene/reactor.png?branch=master)](http://travis-ci.org/jcoene/reactor) [![GoDoc](https://godoc.org/github.com/jcoene/reactor?status.svg)](http://godoc.org/github.com/jcoene/reactor)

Reactor is a "go get"-able library for server-side rendering of React components in Go. If you're on X86_64 Linux or Mac OS X, it should just work.

## Usage

```go
// Import reactor
import "github.com/jcoene/reactor"

// Read your "server-side" javascript bundle, probably compiled with Webpack.
// The only requirement is that it exposes a global render function that receives
// a request returns a response, both encoded as JSON. It will usually look
// something like this:
//
// global.render = (json) => {
//   const req = JSON.parse(json);
//   const component = components(`./${req.name}.jsx`)['default'];
//   const html = ReactDOMServer.renderToString(React.createElement(component, req.props));
//   return JSON.stringify({html: html});
// }
code, _ := ioutil.ReadFile("bundle.js")

// Create a new reactor.Pool with the given code. A pool is a dynamically growing
// group of workers. It supports hot code reloading and scales based on load.
//
// If you only need one worker, you can call reactor.NewWorker.
pool, _ := reactor.NewPool(string(code))

// Make a reactor.Request. Requests contain a component name and optional properties.
// The Properties field is an interface{}, so you can supply a map or any custom type
// that will serialize to JSON easily.
//
// You can also override the Timeout field to supply a custom render timeout.
req := &reactor.Request{
  Name: "MyComponent",
  Props: map[string]interface{}{
    "name": "Alfie",
  },
}

// Render the Request, resulting in a reactor.Response. Responses have an HTML field
// with a string representing the full HTML to be rendered. They also have a Timer
// field indicating how long the render took.
resp, _ := pool.Render(req)

// Do something with resp.HTML
```

## License

MIT License, see [LICENSE](https://github.com/jcoene/reactor/blob/master/LICENSE)

