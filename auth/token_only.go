package auth

import (
	"fmt"
	"slices"
	"strings"
	"time"

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

func (self *TokenOnlyAuthenticator) MakeAuthenticatedWebSocket(cb func(c *websocket.Conn)) func(*fiber.Ctx) error {
	return websocket.New(func(c *websocket.Conn) {
		defer func() {
			closeNormalClosure := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			_ = c.WriteControl(websocket.CloseMessage, closeNormalClosure, time.Now().Add(time.Second))
			c.Close()
			if r := recover(); r != nil {
				fmt.Println("Recovered in f", r)
			}
		}()

		authorization := c.Headers("Authorization")
		if strings.HasPrefix(authorization, "Bearer ") {
			token := strings.TrimPrefix(authorization, "Bearer ")
			if slices.Contains(self.WebSocketTokens, token) {
				cb(c)
			}
		}
	}, websocket.Config{
		WriteBufferSize: 8192,
	})
}
