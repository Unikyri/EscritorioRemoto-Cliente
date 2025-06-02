package entities

import (
	"errors"
	"time"
)

// User representa la entidad de usuario en el cliente
type User struct {
	id        string
	username  string
	role      string
	createdAt time.Time
	lastLogin *time.Time
}

// NewUser crea una nueva instancia de User
func NewUser(id, username, role string) (*User, error) {
	if id == "" {
		return nil, errors.New("user ID cannot be empty")
	}
	if username == "" {
		return nil, errors.New("username cannot be empty")
	}
	if role == "" {
		return nil, errors.New("user role cannot be empty")
	}

	return &User{
		id:        id,
		username:  username,
		role:      role,
		createdAt: time.Now().UTC(),
		lastLogin: nil,
	}, nil
}

// Getters
func (u *User) ID() string {
	return u.id
}

func (u *User) Username() string {
	return u.username
}

func (u *User) Role() string {
	return u.role
}

func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

func (u *User) LastLogin() *time.Time {
	return u.lastLogin
}

// UpdateLastLogin actualiza el timestamp del último login
func (u *User) UpdateLastLogin() {
	now := time.Now().UTC()
	u.lastLogin = &now
}

// IsAdmin verifica si el usuario es administrador
func (u *User) IsAdmin() bool {
	return u.role == "administrator"
}

// IsClient verifica si el usuario es cliente
func (u *User) IsClient() bool {
	return u.role == "client"
}
