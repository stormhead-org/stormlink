package user

import (
	"errors"
	"regexp"

	"github.com/go-playground/validator/v10"
	"stormlink/server/grpc/user/protobuf"
)

var validate = validator.New()

var (
	ErrInvalidInput = errors.New("invalid input")
)

func ValidateRegisterRequest(req *protobuf.RegisterUserRequest) error {
	// Базовая валидация через теги
	err := validate.Struct(req)
	if err != nil {
		return err
	}

	// Проверка на XSS и потенциальные SQL-инъекции
	if hasDangerousInput(req.Name) || hasDangerousInput(req.Email) {
		return errors.New("input contains potentially dangerous characters")
	}

	return nil
}

// Примитивная проверка на XSS/SQL
func hasDangerousInput(input string) bool {
	// Можно усложнить при необходимости
	pattern := regexp.MustCompile(`[<>'"%;()&+]`)
	return pattern.MatchString(input)
}
