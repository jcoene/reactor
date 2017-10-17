package reactor

import (
	"time"
)

// DefaultTimeout is the default Request Timeout value. It can be overridden
// on a per-request basis by supplying a meaningful value in the Request
// Timeout field.
var DefaultTimeout = 5 * time.Second

// Request represents a request to be sent to the server.
type Request struct {
	// Name is the name of the React component you wish to render. It should
	// be meaningful to your server script.
	Name string `json:"name"`

	// Props are the properties to be supplied to the React component, and should
	// be meaningful to your server script.
	Props interface{} `json:"props"`

	// Timeout provides a timeout for the Request. If not supplied, DefaultTimeout
	// will be used.
	Timeout time.Duration `json:"-"`
}

// Response represents a response received from the server.
type Response struct {
	// HTML is the string returned by the server script, typically the HTML
	// resulting from your React rendering.
	HTML string `json:"html,omitempty"`

	// Error is the error returned by the server script (as a string), typically
	// related to some failure to render the component.
	Error string `json:"error,omitempty"`

	// Timer is the runtime of the render request, including all time spent in
	// serialization, routing, and rendering.
	Timer time.Duration `json:"-"`
}
