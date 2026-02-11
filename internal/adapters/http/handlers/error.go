package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rafaelleal24/challenge/internal/core/serviceerrors"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func HandleError(c *gin.Context, err error) {
	var svcErr *serviceerrors.ServiceError
	if errors.As(err, &svcErr) {
		c.JSON(mapKindToHTTP(svcErr.Kind), ErrorResponse{Error: svcErr.Message})
		return
	}

	c.JSON(http.StatusInternalServerError, ErrorResponse{Error: err.Error()})
}

func mapKindToHTTP(kind serviceerrors.ErrorKind) int {
	switch kind {
	case serviceerrors.KindNotFound:
		return http.StatusNotFound
	case serviceerrors.KindConflict:
		return http.StatusConflict
	case serviceerrors.KindUnprocessableEntity:
		return http.StatusUnprocessableEntity
	case serviceerrors.KindInvalidRequest:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
