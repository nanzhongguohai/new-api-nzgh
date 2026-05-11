package router

import (
	"github.com/QuantumNous/new-api/controller"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/gin-gonic/gin"
)

// SetOIDCProviderRouter registers OIDC Provider (Authorization Server) endpoints.
// These are standard OIDC endpoints, not under /api prefix.
func SetOIDCProviderRouter(router *gin.Engine) {
	// OIDC Discovery
	router.GET("/.well-known/openid-configuration", controller.OIDCProviderDiscovery)

	// OIDC Provider endpoints
	oidcGroup := router.Group("/oauth")
	{
		// Authorization endpoint - redirects to login if not authenticated
		oidcGroup.GET("/authorize", controller.OIDCProviderAuthorize)

		// Token endpoint - client credentials / authorization code exchange
		oidcGroup.POST("/token", middleware.CriticalRateLimit(), controller.OIDCProviderToken)

		// UserInfo endpoint - returns user info for valid access token
		oidcGroup.GET("/userinfo", controller.OIDCProviderUserInfo)
		oidcGroup.POST("/userinfo", controller.OIDCProviderUserInfo)

		// JWKS endpoint - public keys for token verification
		oidcGroup.GET("/jwks", controller.OIDCProviderJWKS)
	}
}
