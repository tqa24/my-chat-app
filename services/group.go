package services

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"my-chat-app/models"
	"my-chat-app/repositories"
	"my-chat-app/websockets"
	"strings"

	"github.com/google/uuid"
)

type GroupService interface {
	CreateGroup(name string, creatorID string) (*models.Group, error)
	JoinGroup(groupID string, userID string) error
	JoinGroupByCode(code string, userID string) (*models.Group, error) // Add this
	LeaveGroup(groupID string, userID string) error
	GetGroupByID(id string) (*models.Group, error)
	ListGroupsForUser(userID string) ([]*models.Group, error)
	GetAllGroups() ([]models.Group, error)
}

type groupService struct {
	groupRepo repositories.GroupRepository
	userRepo  repositories.UserRepository // Need UserRepository to get User by ID
	hub       *websockets.Hub             // Inject the websocket hub
}

func NewGroupService(groupRepo repositories.GroupRepository, userRepo repositories.UserRepository, hub *websockets.Hub) GroupService {
	return &groupService{groupRepo, userRepo, hub}
}

func (s *groupService) CreateGroup(name string, creatorID string) (*models.Group, error) {
	// Parse the creatorID to ensure it's a valid UUID.
	_, err := uuid.Parse(creatorID) // Use creatorID directly
	if err != nil {
		log.Printf("CreateGroup: Invalid creatorID: %v, Error: %v", creatorID, err)
		return nil, fmt.Errorf("invalid creator ID: %w", err)
	}

	// Generate a unique code for the group
	code, err := generateUniqueCode() // Implement this function
	if err != nil {
		return nil, fmt.Errorf("error generating group code: %w", err)
	}
	group := &models.Group{
		Name:  name,
		Code:  code,             // Set the code
		Users: []*models.User{}, // Initialize the Users slice
	}

	err = s.groupRepo.Create(group)
	if err != nil {
		log.Printf("CreateGroup: Error creating group: %v", err)
		return nil, fmt.Errorf("error creating group: %w", err)
	}

	// Get the creator user by ID.
	creator, err := s.userRepo.GetByID(creatorID) // Use creatorID directly
	if err != nil {
		log.Printf("CreateGroup: Creator not found: %v, Error: %v", creatorID, err)
		// Rollback: Delete the group if the creator doesn't exist.
		s.groupRepo.Delete(group.ID.String())
		return nil, fmt.Errorf("creator not found: %w", err)
	}

	// Add the creator as a member of the group.
	err = s.groupRepo.AddUser(group, creator)
	if err != nil {
		log.Printf("CreateGroup: Error adding creator to group: %v", err)
		// Rollback: Delete the group if adding the user fails.
		s.groupRepo.Delete(group.ID.String())
		return nil, fmt.Errorf("error adding creator to group: %w", err)
	}
	//Add creator to Hub
	s.hub.AddClientToGroup(creator.ID.String(), group.ID.String())
	return group, nil
}

// Add JoinGroupByCode function
func (s *groupService) JoinGroupByCode(code string, userID string) (*models.Group, error) {
	group, err := s.groupRepo.GetByCode(code) // Implement GetByCode in the repository
	if err != nil {
		return nil, fmt.Errorf("error finding group by code: %w", err)
	}
	if group == nil {
		return nil, errors.New("group not found")
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("error finding user: %w", err)
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	err = s.groupRepo.AddUser(group, user)
	if err != nil {
		return nil, fmt.Errorf("error adding user to group: %w", err)
	}
	s.hub.AddClientToGroup(userID, group.ID.String())

	return group, nil
}
func (s *groupService) JoinGroup(groupID string, userID string) error {
	group, err := s.groupRepo.GetByID(groupID)
	if err != nil {
		return err
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}
	if group == nil || user == nil {
		return errors.New("group or user not found")
	}
	// Add user to websocket hub
	s.hub.AddClientToGroup(userID, groupID)

	return s.groupRepo.AddUser(group, user)
}

func (s *groupService) LeaveGroup(groupID string, userID string) error {
	group, err := s.groupRepo.GetByID(groupID)
	if err != nil {
		return err
	}
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	if group == nil || user == nil {
		return errors.New("group or user not found")
	}
	// Remove user from websocket hub
	s.hub.RemoveClientFromGroup(userID, groupID)
	return s.groupRepo.RemoveUser(group, user)
}
func (s *groupService) GetGroupByID(id string) (*models.Group, error) {
	return s.groupRepo.GetByID(id)
}

func (s *groupService) ListGroupsForUser(userID string) ([]*models.Group, error) {
	log.Printf("ListGroupsForUser: UserID: %s", userID) // Log the userID
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		log.Printf("ListGroupsForUser: Error getting user: %v", err) // Log the error
		return nil, err
	}
	if user == nil {
		log.Printf("ListGroupsForUser: User not found") // Log if user is nil
		return nil, errors.New("user not found")
	}
	groups, err := s.groupRepo.GetGroupsForUser(user) // user, not userID
	if err != nil {
		log.Printf("List groups for user error %v", err)
	}
	return groups, err // Log the error
}

func (s *groupService) GetAllGroups() ([]models.Group, error) {
	return s.groupRepo.GetAll()
}

// Helper function to generate a unique code (you can improve this)
func generateUniqueCode() (string, error) {
	bytes := make([]byte, 6) // 6 bytes will result in 8 base64 characters
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return strings.ReplaceAll(base64.URLEncoding.EncodeToString(bytes), "_", ""), nil

}
