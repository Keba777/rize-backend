package validator

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func Validate(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}
	errs := make(map[string]string)
	for _, e := range err.(validator.ValidationErrors) {
		field := strings.ToLower(e.Field())
		errs[field] = e.Tag()
	}
	return errs
}
