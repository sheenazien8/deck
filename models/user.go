package models

import "gorm.io/gorm"

// User holds the model definition for the User entity.
type User struct {
	gorm.Model
	Name  string
	Email string
}

// GetID returns the user's ID
func (u *User) GetID() uint {
	return u.ID
}
