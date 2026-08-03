package authx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

var (
	ErrUnauthorized = errors.New("authx: unauthorized")
	ErrForbidden    = errors.New("authx: forbidden")
)

type contextKey string

const (
	claimsContextKey contextKey = "authx:claims"
	apiKeyContextKey contextKey = "authx:api_key"
)

// HasRole reports whether claims contain role. A "*" role matches everything.
func (claims *Claims) HasRole(role string) bool {
	if claims == nil {
		return false
	}
	role = strings.TrimSpace(role)
	if role == "" {
		return false
	}
	for _, candidate := range claims.Roles {
		if candidate == "*" || candidate == role {
			return true
		}
	}
	return false
}

// HasAnyRole reports whether claims contain at least one role.
func (claims *Claims) HasAnyRole(roles ...string) bool {
	if len(roles) == 0 {
		return true
	}
	for _, role := range roles {
		if claims.HasRole(role) {
			return true
		}
	}
	return false
}

// HasAllRoles reports whether claims contain all roles.
func (claims *Claims) HasAllRoles(roles ...string) bool {
	for _, role := range roles {
		if !claims.HasRole(role) {
			return false
		}
	}
	return true
}

// HasPermission reports whether claims contain permission. Permission codes
// support the same wildcards as permissionx, including "*" and "resource:*".
func (claims *Claims) HasPermission(permission string) bool {
	if claims == nil {
		return false
	}
	permission = strings.TrimSpace(permission)
	if permission == "" {
		return false
	}
	return hasPermission(claims.Permissions, permission)
}

// HasAnyPermission reports whether claims contain at least one permission.
func (claims *Claims) HasAnyPermission(permissions ...string) bool {
	if len(permissions) == 0 {
		return true
	}
	for _, permission := range permissions {
		if claims.HasPermission(permission) {
			return true
		}
	}
	return false
}

// HasAllPermissions reports whether claims contain all permissions.
func (claims *Claims) HasAllPermissions(permissions ...string) bool {
	for _, permission := range permissions {
		if !claims.HasPermission(permission) {
			return false
		}
	}
	return true
}

// WithClaims stores JWT claims in context.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, claimsContextKey, claims)
}

// ClaimsFromContext returns JWT claims from context.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	if ctx == nil {
		return nil, false
	}
	claims, ok := ctx.Value(claimsContextKey).(*Claims)
	return claims, ok && claims != nil
}

// UserIDFromContext returns the authenticated user id from context.
func UserIDFromContext(ctx context.Context) (string, bool) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims.UserID == "" {
		return "", false
	}
	return claims.UserID, true
}

// TokenExtractor extracts an authentication token from a request.
type TokenExtractor func(*http.Request) (string, bool)

// ErrorHandler writes an authentication or authorization error.
type ErrorHandler func(http.ResponseWriter, *http.Request, error)

type jwtMiddlewareConfig struct {
	extractToken TokenExtractor
	onError      ErrorHandler
}

// JWTMiddlewareOption configures JWTMiddleware.
type JWTMiddlewareOption func(*jwtMiddlewareConfig)

// WithJWTTokenExtractor sets the token extractor used by JWTMiddleware.
func WithJWTTokenExtractor(extractor TokenExtractor) JWTMiddlewareOption {
	return func(config *jwtMiddlewareConfig) {
		if extractor != nil {
			config.extractToken = extractor
		}
	}
}

// WithJWTErrorHandler sets the error handler used by JWTMiddleware.
func WithJWTErrorHandler(handler ErrorHandler) JWTMiddlewareOption {
	return func(config *jwtMiddlewareConfig) {
		if handler != nil {
			config.onError = handler
		}
	}
}

// BearerTokenFromRequest extracts an OAuth2 bearer token from Authorization.
func BearerTokenFromRequest(request *http.Request) (string, bool) {
	if request == nil {
		return "", false
	}
	return ParseBearerToken(request.Header.Get("Authorization"))
}

// JWTMiddleware verifies a bearer JWT and stores claims in request context.
func JWTMiddleware(secret string, opts ...JWTMiddlewareOption) func(http.Handler) http.Handler {
	config := jwtMiddlewareConfig{
		extractToken: BearerTokenFromRequest,
		onError:      defaultAuthErrorHandler,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			token, ok := config.extractToken(request)
			if !ok {
				config.onError(writer, request, ErrUnauthorized)
				return
			}
			claims, err := VerifyJWT(token, secret)
			if err != nil {
				config.onError(writer, request, err)
				return
			}
			next.ServeHTTP(writer, request.WithContext(WithClaims(request.Context(), claims)))
		})
	}
}

// JWTAuth wraps next with JWTMiddleware.
func JWTAuth(secret string, next http.Handler, opts ...JWTMiddlewareOption) http.Handler {
	return JWTMiddleware(secret, opts...)(next)
}

// RequireAnyRole allows requests whose claims contain at least one role.
func RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	return requireClaims(func(claims *Claims) bool {
		return claims.HasAnyRole(roles...)
	})
}

// RequireAllRoles allows requests whose claims contain all roles.
func RequireAllRoles(roles ...string) func(http.Handler) http.Handler {
	return requireClaims(func(claims *Claims) bool {
		return claims.HasAllRoles(roles...)
	})
}

// RequireAnyPermission allows requests whose claims contain one permission.
func RequireAnyPermission(permissions ...string) func(http.Handler) http.Handler {
	return requireClaims(func(claims *Claims) bool {
		return claims.HasAnyPermission(permissions...)
	})
}

// RequireAllPermissions allows requests whose claims contain all permissions.
func RequireAllPermissions(permissions ...string) func(http.Handler) http.Handler {
	return requireClaims(func(claims *Claims) bool {
		return claims.HasAllPermissions(permissions...)
	})
}

func requireClaims(allow func(*Claims) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			claims, ok := ClaimsFromContext(request.Context())
			if !ok {
				defaultAuthErrorHandler(writer, request, ErrUnauthorized)
				return
			}
			if allow != nil && !allow(claims) {
				defaultForbiddenHandler(writer, request, ErrForbidden)
				return
			}
			next.ServeHTTP(writer, request)
		})
	}
}

// APIKeyVerifier validates an API key.
type APIKeyVerifier func(context.Context, string) (bool, error)

type apiKeyMiddlewareConfig struct {
	extractToken TokenExtractor
	onError      ErrorHandler
}

// APIKeyMiddlewareOption configures APIKeyMiddleware.
type APIKeyMiddlewareOption func(*apiKeyMiddlewareConfig)

// WithAPIKeyExtractor sets the API key extractor.
func WithAPIKeyExtractor(extractor TokenExtractor) APIKeyMiddlewareOption {
	return func(config *apiKeyMiddlewareConfig) {
		if extractor != nil {
			config.extractToken = extractor
		}
	}
}

// WithAPIKeyErrorHandler sets the API key error handler.
func WithAPIKeyErrorHandler(handler ErrorHandler) APIKeyMiddlewareOption {
	return func(config *apiKeyMiddlewareConfig) {
		if handler != nil {
			config.onError = handler
		}
	}
}

// APIKeyFromRequest extracts an API key from X-API-Key or bearer auth.
func APIKeyFromRequest(request *http.Request) (string, bool) {
	if request == nil {
		return "", false
	}
	if value := strings.TrimSpace(request.Header.Get("X-API-Key")); value != "" {
		return value, true
	}
	return BearerTokenFromRequest(request)
}

// APIKeyMiddleware verifies an API key and stores it in request context.
func APIKeyMiddleware(verifier APIKeyVerifier, opts ...APIKeyMiddlewareOption) func(http.Handler) http.Handler {
	config := apiKeyMiddlewareConfig{
		extractToken: APIKeyFromRequest,
		onError:      defaultAuthErrorHandler,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if verifier == nil {
				config.onError(writer, request, ErrUnauthorized)
				return
			}
			apiKey, ok := config.extractToken(request)
			if !ok {
				config.onError(writer, request, ErrUnauthorized)
				return
			}
			ok, err := verifier(request.Context(), apiKey)
			if err != nil {
				config.onError(writer, request, err)
				return
			}
			if !ok {
				config.onError(writer, request, ErrUnauthorized)
				return
			}
			next.ServeHTTP(writer, request.WithContext(WithAPIKey(request.Context(), apiKey)))
		})
	}
}

// WithAPIKey stores an API key in context.
func WithAPIKey(ctx context.Context, apiKey string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, apiKeyContextKey, apiKey)
}

// APIKeyFromContext returns an API key from context.
func APIKeyFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	apiKey, ok := ctx.Value(apiKeyContextKey).(string)
	return apiKey, ok && apiKey != ""
}

func defaultAuthErrorHandler(writer http.ResponseWriter, _ *http.Request, _ error) {
	writeJSONError(writer, http.StatusUnauthorized)
}

func defaultForbiddenHandler(writer http.ResponseWriter, _ *http.Request, _ error) {
	writeJSONError(writer, http.StatusForbidden)
}

func writeJSONError(writer http.ResponseWriter, status int) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]string{
		"error": http.StatusText(status),
	})
}
