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

			switch err.Tag() {
			case "required":
				element.Msg = fmt.Sprintf("Kolom '%s' wajib diisi.", element.Field)
			case "min":
				element.Msg = fmt.Sprintf("Kolom '%s' harus memiliki minimal %s karakter/nilai.", element.Field, err.Param())
			case "max":
				element.Msg = fmt.Sprintf("Kolom '%s' harus memiliki maksimal %s karakter/nilai.", element.Field, err.Param())
			case "email":
				element.Msg = "Format email tidak valid."
			case "hasuppercase":
				element.Msg = "Password harus mengandung setidaknya satu huruf kapital."

			case "url":
				element.Msg = fmt.Sprintf("Kolom '%s' harus berupa format URL yang valid.", element.Field)
			case "oneof":
				element.Msg = fmt.Sprintf("Kolom '%s' harus salah satu dari: %s.", element.Field, err.Param())
			default:

				element.Msg = fmt.Sprintf("Kolom '%s' gagal validasi untuk tag '%s'.", element.Field, element.Tag)
			}
			errors = append(errors, &element)
		}
	}
	return errors
}
