package api

import (
	"log"
	"my-chat-app/services"
	"my-chat-app/utils"
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
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}
	log.Printf("Create group: %v", req)

	// Get userID from JWT context (set by middleware)
	userID, exists := c.Get("userID")
	if !exists {
		utils.RespondWithError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Convert to string
	creatorID, ok := userID.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid user ID format")
		return
	}

	group, err := h.groupService.CreateGroup(req.Name, creatorID)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusCreated, group)
}

func (h *GroupHandler) GetGroup(c *gin.Context) {
	groupID := c.Param("id")
	group, err := h.groupService.GetGroupByID(groupID)
	if err != nil {
		utils.RespondWithError(c, http.StatusNotFound, "Group not found")
		return
	}
	c.JSON(http.StatusOK, group)
}

// Join with group ID
func (h *GroupHandler) JoinGroup(c *gin.Context) {
	groupID := c.Param("id")

	// Get userID from JWT context (set by middleware)
	userID, exists := c.Get("userID")
	if !exists {
		utils.RespondWithError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Convert to string
	userIDStr, ok := userID.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid user ID format")
		return
	}

	err := h.groupService.JoinGroup(groupID, userIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Joined group successfully"})
}

// Join with group Code
func (h *GroupHandler) JoinGroupByCode(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload")
		return
	}

	// Get userID from JWT context (set by middleware)
	userID, exists := c.Get("userID")
	if !exists {
		utils.RespondWithError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Convert to string
	userIDStr, ok := userID.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid user ID format")
		return
	}

	group, err := h.groupService.JoinGroupByCode(req.Code, userIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Return the group information. This is important for the frontend.
	c.JSON(http.StatusOK, gin.H{"message": "Joined group successfully", "group": group})
}

func (h *GroupHandler) LeaveGroup(c *gin.Context) {
	groupID := c.Param("id")

	// Get userID from JWT context (set by middleware)
	userID, exists := c.Get("userID")
	if !exists {
		utils.RespondWithError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Convert to string
	userIDStr, ok := userID.(string)
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid user ID format")
		return
	}

	err := h.groupService.LeaveGroup(groupID, userIDStr)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Left group successfully"})
}

func (h *GroupHandler) ListGroupsForUser(c *gin.Context) {
	// The userID can either come from the URL parameter or from the JWT token
	userIDParam := c.Param("id")

	// If no userID in URL, get it from JWT context
	if userIDParam == "" {
		userID, exists := c.Get("userID")
		if !exists {
			utils.RespondWithError(c, http.StatusUnauthorized, "User not authenticated")
			return
		}

		var ok bool
		userIDParam, ok = userID.(string)
		if !ok {
			utils.RespondWithError(c, http.StatusInternalServerError, "Invalid user ID format")
			return
		}
	}

	log.Printf("ListGroupsForUser: UserID from Param: %s", userIDParam)

	groups, err := h.groupService.ListGroupsForUser(userIDParam)
	if err != nil {
		log.Printf("ListGroupsForUser: Error from groupService: %v", err)
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("ListGroupsForUser: Groups found: %+v", groups)
	c.JSON(http.StatusOK, groups)
}

func (h *GroupHandler) GetAllGroups(c *gin.Context) {
	groups, err := h.groupService.GetAllGroups()
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to retrieve groups")
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (h *GroupHandler) GetGroupMembers(c *gin.Context) {
	groupID := c.Param("id")
	members, err := h.groupService.GetGroupMembers(groupID)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to retrieve group members")
		return
	}

	// Convert model to DTO
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
