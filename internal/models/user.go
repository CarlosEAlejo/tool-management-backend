package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	RoleAdministrator = "administrator"
	UserStatusActive  = "active"
)

type User struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email         string             `bson:"email" json:"email"`
	PasswordHash  string             `bson:"password_hash" json:"-"`
	Roles         []string           `bson:"roles" json:"roles"`
	Status        string             `bson:"status" json:"status"`
	EmailVerified bool               `bson:"email_verified" json:"emailVerified"`
	CreatedAt     time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updatedAt"`
	LastLoginAt   *time.Time         `bson:"last_login_at,omitempty" json:"lastLoginAt,omitempty"`
}

type AuthSession struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	UserID     primitive.ObjectID `bson:"user_id"`
	TokenHash  string             `bson:"token_hash"`
	JTI        string             `bson:"jti"`
	ExpiresAt  time.Time          `bson:"expires_at"`
	RevokedAt  *time.Time         `bson:"revoked_at,omitempty"`
	CreatedAt  time.Time          `bson:"created_at"`
	LastUsedAt *time.Time         `bson:"last_used_at,omitempty"`
}

type PublicUser struct {
	ID            string   `json:"id"`
	Email         string   `json:"email"`
	Roles         []string `json:"roles"`
	Status        string   `json:"status"`
	EmailVerified bool     `json:"emailVerified"`
}

func (u User) Public() PublicUser {
	return PublicUser{
		ID:            u.ID.Hex(),
		Email:         u.Email,
		Roles:         append([]string{}, u.Roles...),
		Status:        u.Status,
		EmailVerified: u.EmailVerified,
	}
}
