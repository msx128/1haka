package usersregistry

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID, Email, Name string
	PasswordHash    []byte
}

type UsersRegistry struct {
	mu     sync.RWMutex
	Users  map[string]*User // by email
	JWTKey []byte
}

func NewUsersRegistry(jwtkey []byte) *UsersRegistry {
	return &UsersRegistry{Users: make(map[string]*User), JWTKey: jwtkey}
}

var (
	ErrEmailTaken     = errors.New("email already exists")
	ErrBadCredentials = errors.New("bad email or password")
)

func (r *UsersRegistry) AddUser(email, name, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.Users[email]; ok {
		return ErrEmailTaken
	}

	r.Users[email] = &User{uuid.NewString(), email, name, hash}
	return nil
}

func (r *UsersRegistry) Authenticate(email, password string) (*User, error) {
	r.mu.RLock()
	user, ok := r.Users[email]
	r.mu.RUnlock()

	if !ok || bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(password)) != nil {
		return nil, ErrBadCredentials
	}
	return user, nil
}

func (r *UsersRegistry) IssueToken(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(r.JWTKey)
}

func (r *UsersRegistry) ParseToken(s string) (string, error) {
	tok, err := jwt.ParseWithClaims(s, &jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) { return r.JWTKey, nil },
		jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", nil
	}
	return tok.Claims.(*jwt.RegisteredClaims).Subject, nil
}

// TODO validate email
