package auth

import (
	"context"
	"errors"
	"net/mail"
	"os"
	"strings"
	"time"

	"tool_management_backend/internal/database"
	"tool_management_backend/internal/models"
	"tool_management_backend/internal/notifications"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrRegistrationClosed = errors.New("registration closed")
	ErrEmailAlreadyUsed   = errors.New("email already used")
	ErrInvalidRefresh     = errors.New("invalid refresh token")
	ErrUnauthorized       = errors.New("unauthorized")
)

type Service struct {
	accessSecret  string
	refreshSecret string
	accessTTL     time.Duration
	refreshTTL    time.Duration
	mailer        notifications.VerificationEmailSender
}

type AuthResult struct {
	User         models.PublicUser `json:"user"`
	AccessToken  string            `json:"accessToken"`
	RefreshToken string            `json:"refreshToken"`
	ExpiresIn    int64             `json:"expiresIn"`
}

func NewService() *Service {
	return &Service{
		accessSecret:  envOrDefault("JWT_ACCESS_SECRET", "dev-access-secret-change-me"),
		refreshSecret: envOrDefault("JWT_REFRESH_SECRET", "dev-refresh-secret-change-me"),
		accessTTL:     parseDurationEnv("JWT_ACCESS_TTL", 15*time.Minute),
		refreshTTL:    parseDurationEnv("JWT_REFRESH_TTL", 7*24*time.Hour),
		mailer:        notifications.NoopVerificationEmailSender{},
	}
}

func (s *Service) Register(ctx context.Context, email string, password string, confirmPassword string) (*AuthResult, error) {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}
	if password != confirmPassword {
		return nil, errors.New("passwords do not match")
	}

	users := s.usersCollection()
	adminCount, err := users.CountDocuments(ctx, bson.M{"roles": models.RoleAdministrator})
	if err != nil {
		return nil, err
	}
	if adminCount > 0 {
		return nil, ErrRegistrationClosed
	}

	existingCount, err := users.CountDocuments(ctx, bson.M{"email": normalizedEmail})
	if err != nil {
		return nil, err
	}
	if existingCount > 0 {
		return nil, ErrEmailAlreadyUsed
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user := models.User{
		ID:            primitive.NewObjectID(),
		Email:         normalizedEmail,
		PasswordHash:  string(hash),
		Roles:         []string{models.RoleAdministrator},
		Status:        models.UserStatusActive,
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if _, err := users.InsertOne(ctx, user); err != nil {
		return nil, err
	}

	return s.issueSession(ctx, user)
}

func (s *Service) Login(ctx context.Context, email string, password string) (*AuthResult, error) {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	var user models.User
	if err := s.usersCollection().FindOne(ctx, bson.M{"email": normalizedEmail}).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}

	now := time.Now().UTC()
	_, _ = s.usersCollection().UpdateByID(ctx, user.ID, bson.M{"$set": bson.M{
		"last_login_at": now,
		"updated_at":    now,
	}})
	user.LastLoginAt = &now
	user.UpdatedAt = now

	return s.issueSession(ctx, user)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	claims, err := parseJWT(refreshToken, s.refreshSecret)
	if err != nil {
		return nil, ErrInvalidRefresh
	}
	if tokenType, _ := claims["type"].(string); tokenType != "refresh" {
		return nil, ErrInvalidRefresh
	}

	userIDHex, _ := claims["sub"].(string)
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return nil, ErrInvalidRefresh
	}
	jti, _ := claims["jti"].(string)

	var session models.AuthSession
	if err := s.sessionsCollection().FindOne(ctx, bson.M{
		"user_id":    userID,
		"jti":        jti,
		"token_hash": hashToken(refreshToken),
		"revoked_at": bson.M{"$exists": false},
		"expires_at": bson.M{"$gt": time.Now().UTC()},
	}).Decode(&session); err != nil {
		return nil, ErrInvalidRefresh
	}

	now := time.Now().UTC()
	_, _ = s.sessionsCollection().UpdateByID(ctx, session.ID, bson.M{"$set": bson.M{
		"revoked_at":   now,
		"last_used_at": now,
	}})

	user, err := s.FindUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return s.issueSession(ctx, *user)
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil
	}

	now := time.Now().UTC()
	_, err := s.sessionsCollection().UpdateOne(ctx, bson.M{
		"token_hash": hashToken(refreshToken),
		"revoked_at": bson.M{"$exists": false},
	}, bson.M{"$set": bson.M{
		"revoked_at":   now,
		"last_used_at": now,
	}})
	return err
}

func (s *Service) FindUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	if err := s.usersCollection().FindOne(ctx, bson.M{"_id": id}).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Service) ParseAccessToken(ctx context.Context, token string) (*models.User, error) {
	claims, err := parseJWT(token, s.accessSecret)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if tokenType, _ := claims["type"].(string); tokenType != "access" {
		return nil, ErrUnauthorized
	}

	userIDHex, _ := claims["sub"].(string)
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		return nil, ErrUnauthorized
	}

	user, err := s.FindUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUnauthorized
	}
	return user, nil
}

func (s *Service) issueSession(ctx context.Context, user models.User) (*AuthResult, error) {
	now := time.Now().UTC()
	jti, err := generateRandomHex(32)
	if err != nil {
		return nil, err
	}

	accessExpiresAt := now.Add(s.accessTTL)
	refreshExpiresAt := now.Add(s.refreshTTL)
	baseClaims := map[string]any{
		"sub":   user.ID.Hex(),
		"email": user.Email,
		"roles": user.Roles,
		"iat":   now.Unix(),
	}

	accessClaims := cloneClaims(baseClaims)
	accessClaims["type"] = "access"
	accessClaims["exp"] = accessExpiresAt.Unix()
	accessToken, err := signJWT(accessClaims, s.accessSecret)
	if err != nil {
		return nil, err
	}

	refreshClaims := cloneClaims(baseClaims)
	refreshClaims["type"] = "refresh"
	refreshClaims["exp"] = refreshExpiresAt.Unix()
	refreshClaims["jti"] = jti
	refreshToken, err := signJWT(refreshClaims, s.refreshSecret)
	if err != nil {
		return nil, err
	}

	session := models.AuthSession{
		ID:        primitive.NewObjectID(),
		UserID:    user.ID,
		TokenHash: hashToken(refreshToken),
		JTI:       jti,
		ExpiresAt: refreshExpiresAt,
		CreatedAt: now,
	}
	if _, err := s.sessionsCollection().InsertOne(ctx, session); err != nil {
		return nil, err
	}

	return &AuthResult{
		User:         user.Public(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.accessTTL.Seconds()),
	}, nil
}

func (s *Service) usersCollection() *mongo.Collection {
	return database.GetDatabase().Collection("users")
}

func (s *Service) sessionsCollection() *mongo.Collection {
	return database.GetDatabase().Collection("auth_sessions")
}

func cloneClaims(claims map[string]any) map[string]any {
	cloned := make(map[string]any, len(claims))
	for key, value := range claims {
		cloned[key] = value
	}
	return cloned
}

func normalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", errors.New("invalid email")
	}
	return normalized, nil
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}
