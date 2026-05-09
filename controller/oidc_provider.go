package controller

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// OIDCProviderKeyPair holds the RSA key pair for signing OIDC tokens.
var OIDCProviderKeyPair *rsa.PrivateKey

// OIDCProviderKid is the Key ID used in JWT headers.
const OIDCProviderKid = "new-api-oidc-key"

// InitOIDCProviderKeys initializes the RSA key pair for OIDC token signing.
// If keys are stored in options, loads them; otherwise generates new ones and stores them.
func InitOIDCProviderKeys() error {
	// Try to load existing keys from options
	common.OptionMapRWMutex.RLock()
	storedKey := common.OptionMap["oidc_provider_private_key"]
	common.OptionMapRWMutex.RUnlock()
	if storedKey != "" {
		keyBytes, err := base64.StdEncoding.DecodeString(storedKey)
		if err == nil {
			key, err := jwt.ParseRSAPrivateKeyFromPEM(keyBytes)
			if err == nil {
				OIDCProviderKeyPair = key
				common.SysLog("OIDC Provider: loaded existing RSA key pair")
				return nil
			}
		}
		common.SysLog("OIDC Provider: failed to load existing key pair, generating new ones")
	}

	// Generate new RSA key pair
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key pair: %w", err)
	}
	OIDCProviderKeyPair = key

	// Store the private key in options
	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	encoded := base64.StdEncoding.EncodeToString(pemBytes)
	model.UpdateOption("oidc_provider_private_key", encoded)

	common.SysLog("OIDC Provider: generated new RSA key pair")
	return nil
}

// OIDCProviderDiscovery returns the OIDC discovery document.
func OIDCProviderDiscovery(c *gin.Context) {
	if !system_setting.GetOIDCProviderSettings().Enabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "OIDC Provider is not enabled"})
		return
	}

	baseURL := getOIDCProviderBaseURL(c)
	discovery := gin.H{
		"issuer":                 baseURL,
		"authorization_endpoint": baseURL + "/oauth/authorize",
		"token_endpoint":         baseURL + "/oauth/token",
		"userinfo_endpoint":      baseURL + "/oauth/userinfo",
		"jwks_uri":               baseURL + "/oauth/jwks",
		"scopes_supported":       []string{"openid", "profile", "email"},
		"response_types_supported": []string{
			"code",
			"id_token",
		},
		"grant_types_supported": []string{
			"authorization_code",
			"refresh_token",
		},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"userinfo_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{
			"client_secret_basic",
			"client_secret_post",
		},
		"claims_supported": []string{
			"sub",
			"email",
			"name",
			"preferred_username",
			"picture",
		},
		"code_challenge_methods_supported": []string{"S256"},
	}

	c.JSON(http.StatusOK, discovery)
}

// OIDCProviderAuthorize handles the OIDC authorization request.
func OIDCProviderAuthorize(c *gin.Context) {
	if !system_setting.GetOIDCProviderSettings().Enabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "OIDC Provider is not enabled"})
		return
	}

	// Validate required parameters
	responseType := c.Query("response_type")
	clientId := c.Query("client_id")
	redirectURI := c.Query("redirect_uri")
	scope := c.Query("scope")
	state := c.Query("state")
	codeChallenge := c.Query("code_challenge")

	if clientId == "" || redirectURI == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameters"})
		return
	}

	// Validate client
	client, err := model.GetOIDCClientByClientId(clientId)
	if err != nil {
		if !errors.Is(err, model.ErrOIDCClientNotFound) {
			common.SysError("OIDC Provider: failed to query client: " + err.Error())
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "server_temporarily_unavailable"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid client_id"})
		return
	}

	// Validate redirect URI matches registered one
	if redirectURI != client.RedirectURI {
		c.JSON(http.StatusBadRequest, gin.H{"error": "redirect_uri mismatch"})
		return
	}

	// Check if user is logged in
	session := sessions.Default(c)
	userId := session.Get("id")
	if userId == nil {
		// User not logged in - redirect to login page
		// Save OAuth params in session and redirect to login
		loginURL := "/sign-in?redirect=" + url.QueryEscape(c.Request.URL.RequestURI())
		c.Redirect(http.StatusFound, loginURL)
		return
	}

	// Generate authorization code
	code := common.GetRandomString(32)
	codeHash := sha256.Sum256([]byte(code))
	codeHashStr := fmt.Sprintf("%x", codeHash)

	// Store code data in memory/Redis (TTL: 10 minutes)
	codeData := map[string]interface{}{
		"user_id":        userId,
		"client_id":      clientId,
		"redirect_uri":   redirectURI,
		"code_challenge": codeChallenge,
		"scope":          scope,
		"created_at":     time.Now().Unix(),
	}
	codeDataJSON, _ := json.Marshal(codeData)
	if common.RedisEnabled {
		err = common.RedisSet("oidc_code:"+codeHashStr, string(codeDataJSON), 10*time.Minute)
		if err != nil {
			common.SysError("failed to store OIDC code in Redis: " + err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}
	} else {
		// Store in memory (simple map for non-Redis setups)
		oidcCodeStore.Store(codeHashStr, codeData)
		// Clean up after 10 minutes
		go func() {
			time.Sleep(10 * time.Minute)
			oidcCodeStore.Delete(codeHashStr)
		}()
	}

	// Build redirect URL
	redirectURL, _ := url.Parse(redirectURI)
	query := redirectURL.Query()
	query.Set("code", code)
	if state != "" {
		query.Set("state", state)
	}
	// response_type could be "code" or "code id_token" etc.
	if strings.Contains(responseType, "id_token") {
		// Generate id_token for hybrid flow
		idToken, err := generateOIDCIDToken(userId.(int), clientId, getOIDCProviderBaseURL(c))
		if err != nil {
			common.SysError("failed to generate id_token: " + err.Error())
		} else {
			query.Set("id_token", idToken)
		}
	}
	redirectURL.RawQuery = query.Encode()

	c.Redirect(http.StatusFound, redirectURL.String())
}

// OIDCProviderToken handles the token endpoint (authorization code exchange).
func OIDCProviderToken(c *gin.Context) {
	if !system_setting.GetOIDCProviderSettings().Enabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "OIDC Provider is not enabled"})
		return
	}

	grantType := c.PostForm("grant_type")
	clientId := c.PostForm("client_id")
	clientSecret := c.PostForm("client_secret")

	// Also support Basic auth
	if clientId == "" || clientSecret == "" {
		var ok bool
		clientId, clientSecret, ok = c.Request.BasicAuth()
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_client"})
			return
		}
	}

	// Validate client
	client, err := model.GetOIDCClientByClientId(clientId)
	if err != nil {
		if !errors.Is(err, model.ErrOIDCClientNotFound) {
			common.SysError("OIDC Provider: failed to query client: " + err.Error())
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "server_temporarily_unavailable"})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}
	if client.ClientSecret != clientSecret {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
		return
	}

	switch grantType {
	case "authorization_code":
		handleOIDCTokenExchange(c, client)
	case "refresh_token":
		handleOIDCRefreshToken(c, client)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
	}
}

func handleOIDCTokenExchange(c *gin.Context, client *model.OIDCClient) {
	code := c.PostForm("code")
	redirectURI := c.PostForm("redirect_uri")
	codeVerifier := c.PostForm("code_verifier")

	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "missing code"})
		return
	}

	// Hash the code to look it up
	codeHash := sha256.Sum256([]byte(code))
	codeHashStr := fmt.Sprintf("%x", codeHash)

	var codeDataMap map[string]interface{}
	if common.RedisEnabled {
		val, err := common.RedisGet("oidc_code:" + codeHashStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "code not found or expired"})
			return
		}
		json.Unmarshal([]byte(val), &codeDataMap)
		common.RedisDel("oidc_code:" + codeHashStr)
	} else {
		val, ok := oidcCodeStore.Load(codeHashStr)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "code not found or expired"})
			return
		}
		codeDataMap = val.(map[string]interface{})
		oidcCodeStore.Delete(codeHashStr)
	}

	// Validate code challenge if provided (PKCE)
	if storedChallenge, ok := codeDataMap["code_challenge"].(string); ok && storedChallenge != "" {
		if codeVerifier == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "missing code_verifier"})
			return
		}
		if !verifyPKCE(storedChallenge, codeVerifier) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "code_verifier mismatch"})
			return
		}
	}

	// Validate redirect_uri matches
	if storedRedirectURI, _ := codeDataMap["redirect_uri"].(string); storedRedirectURI != "" && redirectURI != storedRedirectURI {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "redirect_uri mismatch"})
		return
	}

	userId := int(codeDataMap["user_id"].(float64))

	// Generate access token (JWT)
	baseURL := getOIDCProviderBaseURL(c)
	accessToken, err := generateOIDCAccessToken(userId, client.ClientId, baseURL)
	if err != nil {
		common.SysError("failed to generate access token: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// Generate ID token (JWT)
	idToken, err := generateOIDCIDToken(userId, client.ClientId, baseURL)
	if err != nil {
		common.SysError("failed to generate id token: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error"})
		return
	}

	// Generate refresh token
	refreshToken := common.GetRandomString(32)

	c.JSON(http.StatusOK, gin.H{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"id_token":      idToken,
		"refresh_token": refreshToken,
	})
}

func handleOIDCRefreshToken(c *gin.Context, client *model.OIDCClient) {
	// For simplicity, issue a new access token based on the refresh token
	// In production, you'd validate the refresh token against a store
	refreshToken := c.PostForm("refresh_token")
	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "missing refresh_token"})
		return
	}

	// TODO: Validate refresh token from store
	// For now, return an error for unimplemented refresh
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "refresh token not supported yet"})
}

// OIDCProviderUserInfo returns user information for a valid access token.
func OIDCProviderUserInfo(c *gin.Context) {
	if !system_setting.GetOIDCProviderSettings().Enabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "OIDC Provider is not enabled"})
		return
	}

	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token"})
		return
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	// Parse and validate the JWT
	claims, err := validateOIDCToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "error_description": err.Error()})
		return
	}

	userId, err := oidcSubjectToUserID(claims["sub"])
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_token", "error_description": err.Error()})
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	email := getOIDCUserEmail(user)
	userInfo := gin.H{
		"sub":                fmt.Sprintf("%d", user.Id),
		"name":               user.DisplayName,
		"preferred_username": user.Username,
		"email":              email,
	}

	if email != "" {
		userInfo["email_verified"] = true
	}

	// Include model access configuration for lobehub auto-setup
	tokens, tokenErr := model.GetAllUserTokens(user.Id, 0, 1)
	if tokenErr == nil && len(tokens) > 0 {
		token := tokens[0]
		userCache, cacheErr := model.GetUserCache(user.Id)
		if cacheErr == nil {
			var models []string
			if token.ModelLimitsEnabled && token.ModelLimits != "" {
				models = token.GetModelLimits()
			} else {
				groups := service.GetUserUsableGroups(userCache.Group)
				modelSet := make(map[string]bool)
				for group := range groups {
					for _, m := range model.GetGroupEnabledModels(group) {
						if !modelSet[m] {
							modelSet[m] = true
							models = append(models, m)
						}
					}
				}
			}
			baseURL := getOIDCProviderBaseURL(c)
			userInfo["api_url"] = fmt.Sprintf("%s/v1", baseURL)
			userInfo["api_key"] = "sk-" + token.Key
			userInfo["api_models"] = models
		}
	}

	c.JSON(http.StatusOK, userInfo)
}

// OIDCProviderJWKS returns the JWKS document with public keys.
func OIDCProviderJWKS(c *gin.Context) {
	if !system_setting.GetOIDCProviderSettings().Enabled {
		c.JSON(http.StatusNotFound, gin.H{"error": "OIDC Provider is not enabled"})
		return
	}

	if OIDCProviderKeyPair == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "keys not initialized"})
		return
	}

	// Build JWKS
	jwks := gin.H{
		"keys": []gin.H{
			{
				"kty": "RSA",
				"kid": OIDCProviderKid,
				"use": "sig",
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(OIDCProviderKeyPair.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(bigEndianBytes(OIDCProviderKeyPair.E)),
			},
		},
	}

	c.JSON(http.StatusOK, jwks)
}

// generateOIDCAccessToken creates a JWT access token for the given user.
func generateOIDCAccessToken(userId int, clientId string, issuer string) (string, error) {
	if OIDCProviderKeyPair == nil {
		return "", fmt.Errorf("OIDC keys not initialized")
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":       fmt.Sprintf("%d", userId),
		"client_id": clientId,
		"iss":       issuer,
		"iat":       now.Unix(),
		"exp":       now.Add(1 * time.Hour).Unix(),
		"jti":       common.GetRandomString(16),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = OIDCProviderKid

	return token.SignedString(OIDCProviderKeyPair)
}

// generateOIDCIDToken creates a JWT ID token for the given user.
func generateOIDCIDToken(userId int, clientId string, issuer string) (string, error) {
	if OIDCProviderKeyPair == nil {
		return "", fmt.Errorf("OIDC keys not initialized")
	}

	user, err := model.GetUserById(userId, false)
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"sub":                fmt.Sprintf("%d", user.Id),
		"name":               user.DisplayName,
		"preferred_username": user.Username,
		"email":              getOIDCUserEmail(user),
		"email_verified":     true,
		"aud":                clientId,
		"iss":                issuer,
		"iat":                now.Unix(),
		"exp":                now.Add(1 * time.Hour).Unix(),
		// "nonce" would be added here if provided
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = OIDCProviderKid

	return token.SignedString(OIDCProviderKeyPair)
}

func getOIDCProviderBaseURL(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return strings.TrimRight(system_setting.ServerAddress, "/")
	}

	host := firstHeaderValue(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	if host == "" {
		return strings.TrimRight(system_setting.ServerAddress, "/")
	}

	scheme := firstHeaderValue(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		scheme = firstHeaderValue(c.GetHeader("X-Forwarded-Protocol"))
	}
	if scheme == "" && c.Request.TLS != nil {
		scheme = "https"
	}
	if scheme == "" && c.Request.URL != nil && c.Request.URL.Scheme != "" {
		scheme = c.Request.URL.Scheme
	}
	if scheme == "" {
		scheme = "http"
	}

	return strings.TrimRight(strings.ToLower(scheme)+"://"+host, "/")
}

func firstHeaderValue(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	return strings.TrimSpace(parts[0])
}

func getOIDCUserEmail(user *model.User) string {
	if user.Email != "" {
		return user.Email
	}

	localPart := sanitizeEmailLocalPart(user.Username)
	if localPart == "" {
		localPart = fmt.Sprintf("user%d", user.Id)
	}

	return localPart + "@newapi.local"
}

func sanitizeEmailLocalPart(value string) string {
	value = strings.TrimSpace(value)
	var builder strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '%' || r == '+' || r == '-' {
			builder.WriteRune(r)
		} else {
			builder.WriteByte('_')
		}
	}
	return builder.String()
}

func oidcSubjectToUserID(sub interface{}) (int, error) {
	switch value := sub.(type) {
	case string:
		id, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("invalid subject")
		}
		return id, nil
	case float64:
		return int(value), nil
	case int:
		return value, nil
	default:
		return 0, fmt.Errorf("invalid subject")
	}
}

// validateOIDCToken parses and validates a JWT token.
func validateOIDCToken(tokenString string) (jwt.MapClaims, error) {
	if OIDCProviderKeyPair == nil {
		return nil, fmt.Errorf("OIDC keys not initialized")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return &OIDCProviderKeyPair.PublicKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

// verifyPKCE verifies the code_verifier against the code_challenge using S256.
func verifyPKCE(challenge string, verifier string) bool {
	hash := sha256.Sum256([]byte(verifier))
	computedChallenge := base64.RawURLEncoding.EncodeToString(hash[:])
	return challenge == computedChallenge
}

// bigEndianBytes returns the big-endian bytes of an integer.
func bigEndianBytes(n int) []byte {
	var buf []byte
	for n > 0 {
		buf = append([]byte{byte(n & 0xff)}, buf...)
		n >>= 8
	}
	if len(buf) == 0 {
		return []byte{0}
	}
	return buf
}

// oidcCodeStore is a simple in-memory store for OIDC authorization codes.
// Used when Redis is not available.
var oidcCodeStore syncMap

// syncMap is a simple thread-safe map.
type syncMap struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

func (m *syncMap) Load(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

func (m *syncMap) Store(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.data == nil {
		m.data = make(map[string]interface{})
	}
	m.data[key] = value
}

func (m *syncMap) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}
