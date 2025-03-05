package services

import (
	"log"
	"my-chat-app/repositories"
	"time"
)

type CleanupService interface {
	CleanupUnverifiedAccounts() error
	StartCleanupScheduler(interval time.Duration)
}

type cleanupService struct {
	userRepo repositories.UserRepository
}

func NewCleanupService(userRepo repositories.UserRepository) CleanupService {
	return &cleanupService{userRepo}
}

// CleanupUnverifiedAccounts marks unverified accounts with expired OTPs as deleted
func (s *cleanupService) CleanupUnverifiedAccounts() error {
	users, err := s.userRepo.GetUnverifiedWithExpiredOTP()
	if err != nil {
		log.Printf("Error fetching unverified accounts: %v", err)
		return err
	}

	for _, user := range users {
		now := time.Now()
		user.DeletedAt = &now

		log.Printf("Soft deleting unverified account: %s (email: %s)", user.Username, user.Email)

		if err := s.userRepo.Update(user); err != nil {
			log.Printf("Error soft deleting user %s: %v", user.ID, err)
			continue
		}
	}

	return nil
}

// StartCleanupScheduler runs the cleanup process at regular intervals
func (s *cleanupService) StartCleanupScheduler(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := s.CleanupUnverifiedAccounts(); err != nil {
				log.Printf("Error during scheduled cleanup: %v", err)
			}
		}
	}()
	log.Printf("Cleanup scheduler started with interval: %v", interval)
}
