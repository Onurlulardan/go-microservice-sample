package utils

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
)

func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return errors.New("email is required")
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return errors.New("invalid email format")
	}

	return nil
}

func ValidatePhone(phone string) error {
	if phone == "" {
		return nil
	}

	phoneRegex := regexp.MustCompile(`^(\+90|0)?([5][0-9]{9})$`)
	if !phoneRegex.MatchString(phone) {
		return errors.New("invalid phone number format (Turkish format expected)")
	}

	return nil
}

func ValidateRequired(field, fieldName string) error {
	if strings.TrimSpace(field) == "" {
		return errors.New(fieldName + " is required")
	}
	return nil
}

func ValidateLength(field, fieldName string, min, max int) error {
	length := len(strings.TrimSpace(field))
	if length < min {
		return errors.New(fieldName + " must be at least " + string(rune(min)) + " characters")
	}
	if length > max {
		return errors.New(fieldName + " must be at most " + string(rune(max)) + " characters")
	}
	return nil
}
