package errorcustome

import (
	"errors"
	"fmt"
	"maps"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/namnv2496/mocktool/pkg/generated/github.com/namnv/mockTool/pkg/errorcustome"
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
	Details      map[string]string
	TraceId      string
}

func NewError(
	code codes.Code,
	errorCode string,
	errorMessage string,
	details map[string]string,
	args ...interface{},
) error {
	st := status.New(code, fmt.Sprintf(errorMessage, args...))
	stWithDetails, _ := st.WithDetails(&pb.ErrorDetail{
		ErrorCode: errorCode,
		Metadata:  details,
	})

	return stWithDetails.Err()
	// return &ErrorResponse{
	// 	Code:         code,
	// 	ErrorCode:    errorCode,
	// 	ErrorMessage: fmt.Sprintf(errorMessage, args...),
	// 	Details:      details,
	// }
}

func (_self ErrorResponse) Error() string {
	return _self.ErrorMessage
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
	metadataMap := make(map[string]string)
	maps.Copy(metadataMap, errResp.Details)

	st, withDetailErr := st.WithDetails(
		&pb.ErrorDetail{
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
		if d, ok := detail.(*pb.ErrorDetail); ok {
			resp.ErrorCode = d.ErrorCode
			resp.Details = map[string]string{}

			maps.Copy(resp.Details, d.Metadata)
			break
		}
	}

	if v, ok := resp.Details["x-trace-id"]; ok {
		resp.TraceId = v
	}

	return resp
}

// func ConvertToHttpResponse(err error) ErrorResponse {
// 	resp := ErrorResponse{
// 		Success: false,
// 	}

// 	st, ok := status.FromError(err)
// 	if !ok {
// 		resp.Status = http.StatusInternalServerError
// 		resp.ErrorMessage = "internal server error"
// 		return resp
// 	}

// 	resp.Status = runtime.HTTPStatusFromCode(st.Code())
// 	resp.ErrorMessage = st.Message()

// 	for _, detail := range st.Details() {
// 		if d, ok := detail.(*pb.ErrorDetail); ok {
// 			resp.ErrorCode = d.ErrorCode
// 			resp.Details = d.Metadata
// 			break
// 		}
// 	}

// 	if value, exist := resp.Details["x-trace-id"]; exist {
// 		resp.TraceId = value
// 	}
// 	return resp
// }

func isCustomeError(err error) (*ErrorResponse, bool) {
	var errResp ErrorResponse
	if errors.As(err, &errResp) {
		return &errResp, true
	}
	return nil, false
}
