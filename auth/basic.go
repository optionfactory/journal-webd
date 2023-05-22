package auth

import (
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type BasicAuthenticator struct {
	Username        string
	Password        string
	WebSocketTokens []string
}

func MakeBasicAuthenticator(conf *BasicAuthConfig, webSocketTokens []string) *BasicAuthenticator {
	return &BasicAuthenticator{
		Username:        conf.Username,
		Password:        conf.Password,
		WebSocketTokens: webSocketTokens,
	}
}

func (self *BasicAuthenticator) InterceptAssetRequest(cb func(*fiber.Ctx) error) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		c.Append("WWW-Authenticate", "Basic realm=\"Realm\"")
		authorization := c.Get("Authorization")
		if !strings.HasPrefix(authorization, "Basic ") {
			return c.SendStatus(fiber.StatusUnauthorized)
		}
		accessToken := strings.TrimPrefix(authorization, "Basic ")
		if accessToken == "" {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		if accessToken != b64(fmt.Sprintf("%s:%s", self.Username, self.Password)) {
			return c.SendStatus(fiber.StatusForbidden)
		}
		return cb(c)
	}
}

func b64(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

func (self *BasicAuthenticator) InterceptApiCall(c *fiber.Ctx) error {
	authorization := c.Get("Authorization")
	if !strings.HasPrefix(authorization, "Basic ") {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	accessToken := strings.TrimPrefix(authorization, "Basic ")
	if accessToken == "" {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	if accessToken != b64(fmt.Sprintf("%s:%s", self.Username, self.Password)) {
		return c.SendStatus(fiber.StatusForbidden)

	}
	return c.Next()
}

func (self *BasicAuthenticator) MakeAuthenticatedWebSocket(cb func(c *websocket.Conn)) func(*fiber.Ctx) error {
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
				return
			}
		}
		if !strings.HasPrefix(authorization, "Basic ") {
			return
		}
		accessToken := strings.TrimPrefix(authorization, "Basic ")
		if accessToken == "" {
			return
		}

		if accessToken != b64(fmt.Sprintf("%s:%s", self.Username, self.Password)) {
			return
		}

		cb(c)
	}, websocket.Config{
		WriteBufferSize: 8192,
	})
}
