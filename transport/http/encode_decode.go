package http

import "io"

// DecodeFunc converts a serialized request (transport-domain) to a user
// request (business-domain). One straightforward DecodeFunc could be
// something that JSON-decodes the reader to a concrete request type.
type DecodeFunc func(io.Reader) (interface{}, error)

// EncodeFunc converts a user response (business-domain) to a serialized
// response (transport-domain) by encoding the interface to the writer.
type EncodeFunc func(io.Writer, interface{}) error
