package repositories

import (
	"my-chat-app/models"

	"gorm.io/gorm"
)

type GroupRepository interface {
	Create(group *models.Group) error
	GetByID(id string) (*models.Group, error)
	AddUser(group *models.Group, user *models.User) error
	RemoveUser(group *models.Group, user *models.User) error
	GetUsers(group *models.Group) ([]*models.User, error)
	GetGroupsForUser(user *models.User) ([]*models.Group, error) // Corrected
	GetAll() ([]models.Group, error)
	Update(group *models.Group) error
	Delete(id string) error
}

type groupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) GroupRepository {
	return &groupRepository{db}
}

func (r *groupRepository) Create(group *models.Group) error {
	return r.db.Create(group).Error
}

func (r *groupRepository) GetByID(id string) (*models.Group, error) {
	var group models.Group
	err := r.db.Preload("Users").Where("id = ?", id).First(&group).Error //Preload loads the associated users
	return &group, err
}

func (r *groupRepository) AddUser(group *models.Group, user *models.User) error {
	return r.db.Exec("INSERT INTO user_groups (user_id, group_id) VALUES (?, ?)", user.ID, group.ID).Error

}

func (r *groupRepository) RemoveUser(group *models.Group, user *models.User) error {
	return r.db.Model(group).Association("Users").Delete(user)
}

func (r *groupRepository) GetUsers(group *models.Group) ([]*models.User, error) {
	var users []*models.User
	err := r.db.Model(group).Association("Users").Find(&users)
	return users, err
}
func (r *groupRepository) GetAll() ([]models.Group, error) {
	var groups []models.Group
	err := r.db.Preload("Users").Find(&groups).Error // Preload associated users
	return groups, err
}

func (r *groupRepository) Update(group *models.Group) error {
	return r.db.Save(group).Error
}

func (r *groupRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Group{}).Error
}

// Corrected
func (r *groupRepository) GetGroupsForUser(user *models.User) ([]*models.Group, error) {
	var groups []*models.Group
	err := r.db.Model(&models.Group{}). // Use Group model as the base
						Joins("JOIN user_groups ON user_groups.group_id = groups.id"). // Explicit join
						Where("user_groups.user_id = ?", user.ID).                     // Correct condition
						Find(&groups).Error                                            // Find groups
	return groups, err
}
