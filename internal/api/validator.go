package api

import (
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
)

type CustomValidator struct {
	validator *validator.Validate
}

func NewCustomValidator() *CustomValidator {
	v := validator.New()

	// Регистрируем правило: разрешаем буквы, цифры, дефис и подчеркивание
	err := v.RegisterValidation("shortcode", func(fl validator.FieldLevel) bool {
		match, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, fl.Field().String())
		return match
	})
	if err != nil {
		log.Panic().Err(err).Send()
	}

	return &CustomValidator{validator: v}
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validator.Struct(i)
}
