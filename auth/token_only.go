package auth

import (
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type TokenOnlyAuthenticator struct {
	WebSocketTokens []string
}

func MakeTokenOnlyAuthenticator(webSocketTokens []string) *TokenOnlyAuthenticator {
	return &TokenOnlyAuthenticator{
		WebSocketTokens: webSocketTokens,
	}
}

func (self *TokenOnlyAuthenticator) InterceptAssetRequest(cb func(*fiber.Ctx) error) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		panic("should never be called")
	}
}

func (self *TokenOnlyAuthenticator) InterceptApiCall(c *fiber.Ctx) error {
	panic("should never be called")
}

func (self *TokenOnlyAuthenticator) MakeAuthenticatedWebSocket(cb func(c *websocket.Conn) error) func(*fiber.Ctx) error {
	return websocket.New(func(c *websocket.Conn) {
		authorization := c.Headers("Authorization")
		if authorization == "" {
			closeUnauthorized(c)
			return
		}
		if !strings.HasPrefix(authorization, "Bearer ") {
			closeUnauthorized(c)
			return
		}
		token := strings.TrimPrefix(authorization, "Bearer ")
		if !slices.Contains(self.WebSocketTokens, token) {
			closeUnauthorized(c)
			return
		}
		closeAndHandleErrors(c, cb(c))
	}, websocket.Config{
		WriteBufferSize: 8192,
	})
}
