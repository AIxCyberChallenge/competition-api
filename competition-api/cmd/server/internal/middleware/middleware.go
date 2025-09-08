package middleware

import "gorm.io/gorm"

type Handler struct {
	DB *gorm.DB
}
