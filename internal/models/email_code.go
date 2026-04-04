package models

type EmailCode struct {
	Email string `json:"Email" binding:"required,email"`
	Code  string `json:"Code" binding:"required"`
}
