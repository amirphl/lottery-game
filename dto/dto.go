package dto

import (
	"github.com/go-playground/validator/v10"
)

type User struct {
	// TODO change type to uuid
	UUID string `json:"user_id" validate:"required,uuid4"`
	_    struct{}
}

func (u *User) Validate() error {
	return validator.New().Struct(u)
}
