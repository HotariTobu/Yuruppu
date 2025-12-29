package storage

import (
	"errors"
	"testing"

	"google.golang.org/api/googleapi"
)

func TestIsPreconditionFailed_True(t *testing.T) {
	err := &googleapi.Error{Code: httpPreconditionFailed}
	if !isPreconditionFailed(err) {
		t.Error("expected true for HTTP 412 error")
	}
}

func TestIsPreconditionFailed_False_OtherCode(t *testing.T) {
	err := &googleapi.Error{Code: 500}
	if isPreconditionFailed(err) {
		t.Error("expected false for HTTP 500 error")
	}
}

func TestIsPreconditionFailed_False_NonAPIError(t *testing.T) {
	err := errors.New("some error")
	if isPreconditionFailed(err) {
		t.Error("expected false for non-API error")
	}
}

func TestIsPreconditionFailed_False_Nil(t *testing.T) {
	if isPreconditionFailed(nil) {
		t.Error("expected false for nil error")
	}
}

func TestIsPreconditionFailed_WrappedError(t *testing.T) {
	apiErr := &googleapi.Error{Code: httpPreconditionFailed}
	wrappedErr := errors.Join(errors.New("wrapper"), apiErr)
	if !isPreconditionFailed(wrappedErr) {
		t.Error("expected true for wrapped HTTP 412 error")
	}
}
