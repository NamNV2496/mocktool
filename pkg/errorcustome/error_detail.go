package errorcustome

type ErrorDetail struct {
	ErrorCode string
	Metadata  map[string]any
}

// ProtoMessage is a dummy implementation of protoiface.MessageV1
func (e *ErrorDetail) ProtoMessage() {}

func (e *ErrorDetail) Reset()         {}
func (e *ErrorDetail) String() string { return e.ErrorCode }
