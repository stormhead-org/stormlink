package user

import (
	"errors"
	"regexp"

	"stormlink/server/grpc/user/protobuf"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

var (
	ErrInvalidInput = errors.New("invalid input")
)

// Примитивная проверка на XSS/SQL
func hasDangerousInput(input string) bool {
	pattern := regexp.MustCompile(`[<>'"%;()&+]`)
	return pattern.MatchString(input)
}

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