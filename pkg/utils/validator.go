package util

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New()

	Validate.RegisterValidation("hasuppercase", validateHasUppercase)
}

func validateHasUppercase(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	return regexp.MustCompile(`[A-Z]`).MatchString(password)
}

type ErrorResponse struct {
	Field string `json:"field"`
	Tag   string `json:"tag"`
	Msg   string `json:"message"`
}

func ValidateStruct(s interface{}) []*ErrorResponse {
	var errors []*ErrorResponse
	err := Validate.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var element ErrorResponse
			element.Field = err.Field()
			element.Tag = err.Tag()
			element.Msg = fmt.Sprintf("Field '%s' failed validation for tag '%s'", err.Field(), err.Tag())

			if err.Tag() == "hasuppercase" {
				element.Msg = "Password must contain at least one uppercase letter."
			} else if err.Tag() == "email" {
				element.Msg = "Invalid email format."
			} else if err.Tag() == "required" {
				element.Msg = fmt.Sprintf("Field '%s' is required.", err.Field())
			} else if err.Tag() == "min" {
				element.Msg = fmt.Sprintf("Field '%s' must be at least %s characters long.", err.Field(), err.Param())
			} else if err.Tag() == "max" {
				element.Msg = fmt.Sprintf("Field '%s' must be at most %s characters long.", err.Field(), err.Param())
			}
			errors = append(errors, &element)
		}
	}
	return errors
}
