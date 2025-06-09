package models

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"
)

func TestW3CErr(t *testing.T) {
	g := NewWithT(t)
	err := errors.New("test error")

	got := NewW3CErr(123, "test", err)

	g.Expect(got).To(Equal(&W3CError{
		code: 123,
		err:  err,
		Value: ErrorBody{
			Error:      "test",
			Message:    "test error",
			StackTrace: "test error",
		},
	}))

	g.Expect(got.Code()).To(Equal(123))
	g.Expect(got.Error()).To(Equal("test error"))
	g.Expect(got.Unwrap()).To(BeIdenticalTo(err))
}

func TestErrorMessage(t *testing.T) {
	g := NewWithT(t)
	err := errors.New("test error")

	got := NewErrorMessage(123, err)

	g.Expect(got.Code()).To(Equal(123))
	g.Expect(got.Error()).To(Equal("test error"))
	g.Expect(got.Unwrap()).To(BeIdenticalTo(err))
}
