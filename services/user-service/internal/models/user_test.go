package models

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateUser_Single(t *testing.T) {
	// Reset state before test
	ResetUsers()

	user := CreateUser("John Doe", "john@example.com")

	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "John Doe", user.Name)
	assert.Equal(t, "john@example.com", user.Email)
	assert.Equal(t, 1, len(users))
}

func TestCreateUser_MultiUser(t *testing.T) {
	// Reset state before test
	ResetUsers()

	user1 := CreateUser("User1", "user1@example.com")
	user2 := CreateUser("User2", "user2@example.com")

	assert.Equal(t, 1, user1.ID)
	assert.Equal(t, 2, user2.ID)
	assert.Equal(t, 2, len(users))
}

func TestCreateUser_Async(t *testing.T) {
	// Reset state before test
	ResetUsers()

	const numGoroutines = 10
	var wg sync.WaitGroup
	newUserList := make([]User, numGoroutines)

	// Create users concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			newUserList[index] = CreateUser("User", "user@example.com")
		}(i)
	}

	wg.Wait()

	// Verify all users have unique IDs
	// - The size of the set will be "numGoroutines" if all unique IDs
	// - If there is duplicate idSet[i] will be true
	idSet := make(map[int]bool)
	for _, user := range newUserList {
		assert.False(t, idSet[user.ID], "Duplicate ID found: %d", user.ID)
		idSet[user.ID] = true
	}

	assert.Equal(t, numGoroutines, len(users))
	assert.Equal(t, numGoroutines, len(idSet))
}

func TestCreateUser_Async2(t *testing.T) {
	// Reset state before test
	ResetUsers()

	done := make(chan bool)
	const numGoroutines = 10

	newUserList := make([]User, numGoroutines)

	// Create users concurrently
	for i := range numGoroutines {
		go func(index int) {
			newUserList[index] = CreateUser("User", "user@example.com")
			done <- true
		}(i)
	}
	for range numGoroutines {
		<-done
	}

	// Verify all users have unique IDs
	// - The size of the set will be "numGoroutines" if all unique IDs
	// - If there is duplicate idSet[i] will be true
	idSet := make(map[int]bool)
	for _, user := range newUserList {
		assert.False(t, idSet[user.ID], "Duplicate ID found: %d", user.ID)
		idSet[user.ID] = true
	}

	assert.Equal(t, numGoroutines, len(users))
	assert.Equal(t, numGoroutines, len(idSet))
}
