package errorcustome

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewError(t *testing.T) {
	tests := []struct {
		name      string
		code      codes.Code
		errorCode string
		message   string
		metadata  map[string]string
		args      []interface{}
	}{
		{
			name:      "internal error with args",
			code:      codes.Internal,
			errorCode: "ERR.001",
			message:   "Internal server error: %s",
			metadata:  map[string]string{"trace_id": "123"},
			args:      []interface{}{"database connection failed"},
		},
		{
			name:      "not found error",
			code:      codes.NotFound,
			errorCode: "ERR.002",
			message:   "Resource not found: %s",
			metadata:  map[string]string{"resource": "user"},
			args:      []interface{}{"user-123"},
		},
		{
			name:      "invalid argument error",
			code:      codes.InvalidArgument,
			errorCode: "ERR.003",
			message:   "Invalid input",
			metadata:  nil,
			args:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewError(tt.code, tt.errorCode, tt.message, tt.metadata, tt.args...)

			assert.NotNil(t, err)
			assert.Error(t, err)

			// Check that it's a gRPC status error
			st, ok := status.FromError(err)
			assert.True(t, ok, "Error should be a gRPC status error")
			assert.Equal(t, tt.code, st.Code())
		})
	}
}

func TestErrorResponse_Error(t *testing.T) {
	errResp := ErrorResponse{
		Success:      false,
		HttpStatus:   500,
		GrpcCode:     codes.Internal,
		ErrorCode:    "ERR.001",
		ErrorMessage: "Something went wrong",
		TraceId:      "trace-123",
	}

	errorString := errResp.Error()
	assert.Equal(t, "Something went wrong", errorString)
}

func TestWrapErrorResponse(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		err            error
		expectedCode   codes.Code
		checkDetails   bool
	}{
		{
			name: "grpc status error with details",
			err: NewError(
				codes.NotFound,
				"ERR.002",
				"Not found",
				map[string]string{"key": "value"},
			),
			expectedCode: codes.NotFound,
			checkDetails: true,
		},
		{
			name:         "regular error",
			err:          assert.AnError,
			expectedCode: codes.Unknown,
			checkDetails: false,
		},
		{
			name:         "nil error",
			err:          nil,
			expectedCode: codes.OK,
			checkDetails: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := WrapErrorResponse(ctx, tt.err)

			assert.NotNil(t, response)
			assert.Equal(t, tt.expectedCode, response.GrpcCode)
			assert.False(t, response.Success)

			if tt.err == nil {
				assert.Equal(t, codes.OK, response.GrpcCode)
			}
		})
	}
}

func TestNewError_WithFormatting(t *testing.T) {
	err := NewError(
		codes.Internal,
		"ERR.FORMAT",
		"Error: %s, Code: %d, Value: %v",
		nil,
		"test error",
		404,
		true,
	)

	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Contains(t, st.Message(), "test error")
	assert.Contains(t, st.Message(), "404")
	assert.Contains(t, st.Message(), "true")
}

func TestWrapErrorResponse_WithMetadata(t *testing.T) {
	ctx := context.Background()

	metadata := map[string]string{
		"field1":     "value1",
		"x-trace-id": "trace-456",
	}

	err := NewError(
		codes.Internal,
		"ERR.TEST",
		"Test error message",
		metadata,
	)

	response := WrapErrorResponse(ctx, err)

	assert.False(t, response.Success)
	assert.Equal(t, codes.Internal, response.GrpcCode)
	assert.Equal(t, "ERR.TEST", response.ErrorCode)
	assert.Equal(t, "Test error message", response.ErrorMessage)
	assert.Equal(t, "trace-456", response.TraceId)
	assert.NotNil(t, response.Details)
}

func TestErrorResponse_Fields(t *testing.T) {
	metadata := map[string]string{
		"field1": "value1",
		"field2": "value2",
	}

	errResp := ErrorResponse{
		Success:      false,
		HttpStatus:   500,
		GrpcCode:     codes.Internal,
		ErrorCode:    "ERR.TEST",
		ErrorMessage: "Test error message",
		Details:      metadata,
		TraceId:      "trace-456",
	}

	assert.False(t, errResp.Success)
	assert.Equal(t, 500, errResp.HttpStatus)
	assert.Equal(t, codes.Internal, errResp.GrpcCode)
	assert.Equal(t, "ERR.TEST", errResp.ErrorCode)
	assert.Equal(t, "Test error message", errResp.ErrorMessage)
	assert.Equal(t, metadata, errResp.Details)
	assert.Equal(t, "trace-456", errResp.TraceId)
}
