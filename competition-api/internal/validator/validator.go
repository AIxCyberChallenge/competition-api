package validator

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Echo compatible validator with proper tag semantics
type CustomValidator struct {
	validator *validator.Validate
}

func (cv *CustomValidator) Validate(i any) error {
	return cv.validator.Struct(i)
}

func Create() CustomValidator {
	validate := validator.New()
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		paramName := strings.SplitN(field.Tag.Get("param"), ",", 2)[0]
		if paramName != "" {
			return paramName
		}

		jsonName := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if jsonName == "-" {
			return ""
		}
		if jsonName == "-," {
			return "-"
		}
		return jsonName
	})

	return CustomValidator{validator: validate}
}
