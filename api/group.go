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

func (h *GroupHandler) JoinGroup(c *gin.Context) {
	groupID := c.Param("id")
	// userID := c.GetString("userID") // Get user ID from JWT middleware -- REMOVE THIS
	var req struct {
		UserID string `json:"user_id"` // Get user_id from the request body
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}
	userID := req.UserID // Get the user_id from the request

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
	// In a real scenario, you would also verify that the requesting user
	// has permission to see this user's groups (often, it's themselves).

	groups, err := h.groupService.ListGroupsForUser(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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
