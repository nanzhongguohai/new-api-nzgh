package controller

import (
	"fmt"
	"net/http"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// UserModelsConfig returns the user's model access configuration for lobehub integration.
// Returns: { url, key, models, username, display_name }
// This endpoint is designed to be called by lobehub after SSO login to auto-configure AI provider settings.
func UserModelsConfig(c *gin.Context) {
	id := c.GetInt("id")
	if id == 0 {
		// Try session-based auth
		session := sessions.Default(c)
		sessionId := session.Get("id")
		if sessionId == nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"message": "not authenticated",
			})
			return
		}
		id = sessionId.(int)
	}

	user, err := model.GetUserById(id, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "user not found",
		})
		return
	}

	// Get user's tokens
	tokens, err := model.GetAllUserTokens(user.Id, 0, 1)
	if err != nil || len(tokens) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "no API tokens found - please create one in New-API console",
		})
		return
	}

	// Use the first enabled token
	token := tokens[0]

	// Get user's available models
	userCache, err := model.GetUserCache(user.Id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "failed to get user cache",
		})
		return
	}

	// Resolve models (same logic as GetUserModels but returning the full config)
	var models []string
	if token.ModelLimitsEnabled && token.ModelLimits != "" {
		models = token.GetModelLimits()
	} else {
		// Get all models available to the user's groups
		groups := service.GetUserUsableGroups(userCache.Group)
		modelSet := make(map[string]bool)
		for group := range groups {
			for _, g := range model.GetGroupEnabledModels(group) {
				if !modelSet[g] {
					modelSet[g] = true
					models = append(models, g)
				}
			}
		}
	}

	// Build response
	baseURL := system_setting.ServerAddress

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"url":          fmt.Sprintf("%s/v1", baseURL),
		"key":          "sk-" + token.Key,
		"models":       models,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"email":        user.Email,
	})
}
