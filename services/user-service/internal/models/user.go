package models

import "sync"

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// In-memory storage for MVP
var (
	users  = make(map[int]User)
	nextID = 1
	mu     sync.Mutex
)

// Helpers
// Reset user info for testing
func ResetUsers() {
	mu.Lock()
	defer mu.Unlock()

	users = make(map[int]User)
	nextID = 1
}

// GetAllUsers returns all users
func GetAllUsers() []User {
	userList := make([]User, 0, len(users))
	for _, user := range users {
		userList = append(userList, user)
	}
	return userList
}

// CreateUser adds a new user and returns it
func CreateUser(name, email string) User {
	mu.Lock()
	defer mu.Unlock()

	user := User{
		ID:    nextID,
		Name:  name,
		Email: email,
	}
	users[nextID] = user
	nextID++
	return user
}
