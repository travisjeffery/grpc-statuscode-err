package grpcerr

import (
	"context"
	"reflect"
	"strconv"

	"github.com/go-kit/kit/endpoint"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type statusCoder interface {
	StatusCode() int
}

type withStatusCode struct {
	error
	statusCode int
}

func (w *withStatusCode) StatusCode() int { return w.statusCode }

func wrap(err error, statusCode int) error {
	if err == nil {
		return nil
	}
	return &withStatusCode{error: err, statusCode: statusCode}
}

const (
	statusCodeKey = "grpcerr_status_code"
)

// ClientErrMiddleware is used on the client-requests.
func ClientErrMiddleware(errType interface{}) endpoint.Middleware {
	et := reflect.TypeOf(errType)
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			response, err = next(ctx, request)
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return response, err
			}
			statusCodeVal, ok := md[statusCodeKey]
			if !ok {
				return response, err
			}
			statusCode, err := strconv.Atoi(statusCodeVal[0])
			if err != nil {
				return response, err
			}
			if _, ok := reflect.New(et).Interface().(statusCoder); ok {
				return response, wrap(err, statusCode)
			}
			return response, err
		}
	}
}

// ServerErrMiddleware is used on the server-responses.
func ServerErrMiddleware(errType interface{}) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			response, err = next(ctx, request)
			if errsc, ok := err.(statusCoder); ok {
				header := metadata.Pairs(statusCodeKey, strconv.Itoa(errsc.StatusCode()))
				grpc.SendHeader(ctx, header)
			}
			return response, err
		}
	}
}
