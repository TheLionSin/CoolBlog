package controllers

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

func validationErrors(err error) map[string]string {
	out := map[string]string{}

	var ves validator.ValidationErrors
	ok := errors.As(err, &ves)
	if !ok {
		out["_"] = "validation error"
		return out
	}

	for _, e := range ves {
		out[e.Field()] = fmt.Sprintf("не проходит '%s'", e.Tag())
	}
	return out
}
