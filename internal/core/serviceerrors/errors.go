package serviceerrors

import "errors"

type ErrorKind int

const (
	KindNotFound ErrorKind = iota
	KindConflict
	KindUnprocessableEntity
	KindInvalidRequest
)

func IsOfKind(err error, kind ErrorKind) bool {
	var svcErr *ServiceError
	if errors.As(err, &svcErr) {
		return svcErr.Kind == kind
	}
	return false
}

type ServiceError struct {
	Kind    ErrorKind
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

func NewNotFoundError(message string) *ServiceError {
	return &ServiceError{Kind: KindNotFound, Message: message}
}

func NewConflictError(message string) *ServiceError {
	return &ServiceError{Kind: KindConflict, Message: message}
}

func NewUnprocessableEntityError(message string) *ServiceError {
	return &ServiceError{Kind: KindUnprocessableEntity, Message: message}
}

func NewInvalidRequestError(message string) *ServiceError {
	return &ServiceError{Kind: KindInvalidRequest, Message: message}
}
