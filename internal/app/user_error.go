package app

import (
	"fmt"
	"strings"
)

type UserError struct {
	cause       error
	UserMessage string
}

func NewUserError(userMessage string) *UserError {
	return &UserError{UserMessage: userMessage}
}

func (err *UserError) WithCause(cause error) *UserError {
	err.cause = cause
	return err
}

func (err *UserError) Error() string {
	msg := &strings.Builder{}
	_, _ = fmt.Fprintf(msg, "user error with message=%q", err.UserMessage)
	if err.cause != nil {
		_, _ = fmt.Fprintf(msg, " and cause err=%q", err.cause.Error())
	}
	return msg.String()
}
