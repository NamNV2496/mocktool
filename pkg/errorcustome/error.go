package errorcustome

import (
	"fmt"
	"maps"

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
	Success      bool              `json:"success"`
	HttpStatus   int               `json:"http_status,omitempty"`
	GrpcCode     codes.Code        `json:"grpc_code"`
	ErrorCode    string            `json:"error_code,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	Details      map[string]string `json:"details,omitempty"`
	TraceId      string            `json:"trace_id,omitempty"`
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
}

func (_self ErrorResponse) Error() string {
	return _self.ErrorMessage
}

// func ConvertToGrpc(err error) error {
// 	if err == nil {
// 		return nil
// 	}

// 	errResp, ok := isCustomeError(err)
// 	if !ok {
// 		return status.Errorf(codes.Internal, "Internal server error")
// 	}
// 	st := status.New(errResp.GrpcCode, errResp.ErrorMessage)
// 	// Convert metadata to string map
// 	metadataMap := make(map[string]string)
// 	maps.Copy(metadataMap, errResp.Details)

// 	st, withDetailErr := st.WithDetails(
// 		&pb.ErrorDetail{
// 			ErrorCode: errResp.ErrorCode,
// 			Metadata:  metadataMap,
// 		},
// 	)
// 	if withDetailErr != nil {
// 		return status.Errorf(codes.Internal, "failed to attach error detail")
// 	}

// 	return st.Err()
// }

func ConvertToHttpResponse(err error) ErrorResponse {
	resp := ErrorResponse{
		Success: false,
	}

	st, _ := status.FromError(err)
	resp.HttpStatus = runtime.HTTPStatusFromCode(st.Code())
	resp.ErrorMessage = st.Message()
	resp.GrpcCode = st.Code()

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

// func isCustomeError(err error) (*ErrorResponse, bool) {
// 	var errResp ErrorResponse
// 	if errors.As(err, &errResp) {
// 		return &errResp, true
// 	}
// 	return nil, false
// }

func WrapErrorResponse(err error) ErrorResponse {
	resp := ErrorResponse{
		Success: false,
	}
	st, _ := status.FromError(err)

	resp.HttpStatus = runtime.HTTPStatusFromCode(st.Code())
	resp.ErrorMessage = st.Message()
	resp.GrpcCode = st.Code()

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
