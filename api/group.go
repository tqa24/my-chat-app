package api

import (
	"log"
	"my-chat-app/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GroupHandler struct {
	groupService services.GroupService
}

func NewGroupHandler(groupService services.GroupService) *GroupHandler {
	return &GroupHandler{groupService}
}

func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var req struct {
		Name      string `json:"name"`
		CreatorID string `json:"creator_id"` // Temporary: Get creator ID from request
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	log.Printf("Create group: %v", req)
	// In a real application, you'd get the user ID from the JWT.  For now, we use the request parameter.
	// creatorID := c.GetString("userID") // <- REMOVE THIS LINE (for now)
	creatorID := req.CreatorID // Use the creator_id from the request (TEMPORARY)

	if creatorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	group, err := h.groupService.CreateGroup(req.Name, creatorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

func (h *GroupHandler) GetGroup(c *gin.Context) {
	groupID := c.Param("id")
	group, err := h.groupService.GetGroupByID(groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}
	c.JSON(http.StatusOK, group)
}

// Join with group ID
func (h *GroupHandler) JoinGroup(c *gin.Context) {
	groupID := c.Param("id")
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	userID := req.UserID
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	err := h.groupService.JoinGroup(groupID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Joined group successfully"})
}

// Join with group Code
func (h *GroupHandler) JoinGroupByCode(c *gin.Context) {
	var req struct {
		Code   string `json:"code"`
		UserID string `json:"user_id"` // Temporary: Get user ID from request
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// userID := c.GetString("userID") // TODO: Replace with JWT
	userID := req.UserID
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	group, err := h.groupService.JoinGroupByCode(req.Code, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return the group information.  This is important for the frontend.
	c.JSON(http.StatusOK, gin.H{"message": "Joined group successfully", "group": group})
}

func (h *GroupHandler) LeaveGroup(c *gin.Context) {
	groupID := c.Param("id")
	//	userID := c.GetString("userID") // Get user ID from JWT middleware -- REMOVE THIS

	var req struct {
		UserID string `json:"user_id"` // Get user_id from the request body
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	userID := req.UserID // Get the user_id from the request.
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err := h.groupService.LeaveGroup(groupID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Left group successfully"})
}

func (h *GroupHandler) ListGroupsForUser(c *gin.Context) {
	userID := c.Param("id")
	log.Printf("ListGroupsForUser: UserID from Param: %s", userID) // Log the userID

	// In a real scenario, you would also verify that the requesting user
	// has permission to see this user's groups (often, it's themselves).

	groups, err := h.groupService.ListGroupsForUser(userID)
	if err != nil {
		log.Printf("ListGroupsForUser: Error from groupService: %v", err) // Log the error
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("ListGroupsForUser: Groups found: %+v", groups) // Log the groups
	c.JSON(http.StatusOK, groups)
}
func (h *GroupHandler) GetAllGroups(c *gin.Context) {
	groups, err := h.groupService.GetAllGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve groups"})
		return
	}
	c.JSON(http.StatusOK, groups)
}
func (h *GroupHandler) GetGroupMembers(c *gin.Context) {
	groupID := c.Param("id")
	members, err := h.groupService.GetGroupMembers(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve group members"})
		return
	}
	//Convert model to DTO.
	type UserResponse struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	var memberResponses []UserResponse
	for _, member := range members {
		memberResponses = append(memberResponses, UserResponse{
			ID:       member.ID.String(),
			Username: member.Username,
			Email:    member.Email,
		})
	}

	c.JSON(http.StatusOK, memberResponses)
}
