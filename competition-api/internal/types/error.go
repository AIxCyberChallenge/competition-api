package types

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

type (
	Error struct {
		Fields  *map[string]string `json:"fields,omitempty" validate:"optional"`
		Message string             `json:"message"          validate:"required"`
	}
)

func StringError(err string) Error {
	return Error{Message: err}
}

func ValidationError(err error) Error {
	validationErrors, ok := err.(validator.ValidationErrors)
	if ok {
		errorMap := make(map[string]string)
		for _, fieldError := range validationErrors {
			errorMap[fieldError.Field()] = fmt.Sprintf(
				"Failed to validate while checking condition: %s",
				fieldError.Tag(),
			)
		}

		return Error{Message: "validation error", Fields: &errorMap}
	}

	return Error{Message: "validation error"}
}
