package errorcustome

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// type IErrorResponse interface {
// 	ConvertToGrpc(err error) error
// 	ConvertToHttpResponse(err error) ErrorResponse
// }

type ErrorResponse struct {
	Success      bool
	Status       int
	Code         codes.Code
	ErrorCode    string
	ErrorMessage string
	Metadata     map[string]any
}

func NewError(
	code codes.Code,
	errorCode string,
	errorMessage string,
	metadata map[string]any,
	args ...interface{},
) *ErrorResponse {

	return &ErrorResponse{
		Code:         code,
		ErrorCode:    errorCode,
		ErrorMessage: fmt.Sprintf(errorMessage, args...),
		Metadata:     metadata,
	}
}

func ConvertToGrpc(err error) error {
	if err == nil {
		return nil
	}

	errResp, ok := isCustomeError(err)
	if !ok {
		return status.Errorf(codes.Internal, "Internal server error")
	}

	st := status.New(errResp.Code, errResp.ErrorMessage)

	// Convert metadata to string map
	metadataMap := make(map[string]any)
	for k, v := range errResp.Metadata {
		metadataMap[k] = v
	}

	st, withDetailErr := st.WithDetails(
		&ErrorDetail{
			ErrorCode: errResp.ErrorCode,
			Metadata:  metadataMap,
		},
	)
	if withDetailErr != nil {
		return status.Errorf(codes.Internal, "failed to attach error detail")
	}

	return st.Err()
}

func ConvertToHttpResponse(err error) ErrorResponse {
	resp := ErrorResponse{
		Success: false,
	}

	st, ok := status.FromError(err)
	if !ok {
		resp.Status = http.StatusInternalServerError
		resp.ErrorMessage = "internal server error"
		return resp
	}

	resp.Status = runtime.HTTPStatusFromCode(st.Code())
	resp.ErrorMessage = st.Message()

	for _, detail := range st.Details() {
		if d, ok := detail.(*ErrorDetail); ok {
			resp.ErrorCode = d.ErrorCode
			resp.Metadata = d.Metadata
			break
		}
	}

	return resp
}

func isCustomeError(err error) (*ErrorResponse, bool) {
	var errResp ErrorResponse
	if errors.As(err, &errResp) {
		return &errResp, true
	}
	return nil, false
}
