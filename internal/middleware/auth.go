package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	authorizationHeader = "Authorization"
	bearerPrefix        = "Bearer "
	contextUserID       = "userID"
)

type TokenVerifier interface {
	Verify(token string) (string, error)
}

func Auth(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader(authorizationHeader)
		if !strings.HasPrefix(header, bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token de acesso ausente"})
			return
		}
		userID, err := verifier.Verify(strings.TrimPrefix(header, bearerPrefix))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token de acesso inválido"})
			return
		}
		c.Set(contextUserID, userID)
		c.Next()
	}
}
