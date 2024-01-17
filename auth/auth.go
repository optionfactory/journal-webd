package auth

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type UiAuthConfig struct {
	AuthType                string                  `json:"auth_type"` //basic, authorization-code
	AuthorizationCodeConfig AuthorizationCodeConfig `json:"authorization_code_config"`
	BasicAuthConfig         BasicAuthConfig         `json:"basic_auth_config"`
}

type Authenticator interface {
	InterceptAssetRequest(cb func(*fiber.Ctx) error) func(*fiber.Ctx) error
	InterceptApiCall(c *fiber.Ctx) error
	MakeAuthenticatedWebSocket(cb func(c *websocket.Conn) error) func(*fiber.Ctx) error
}

type WebsocketAuthorizationRequest struct {
	Authorization string `json:"authorization"`
}

type AuthorizationCodeConfig struct {
	ClientId     string `json:"client_id"`
	RealmBaseUrl string `json:"realm_base_url"`
	PublicKey    string `json:"public_key"`
}

type BasicAuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func MakeAuthenticator(authConfig *UiAuthConfig, webSocketTokens []string) (Authenticator, error) {
	if authConfig == nil {
		return MakeTokenOnlyAuthenticator(webSocketTokens), nil
	}
	switch {
	case authConfig.AuthType == "basic":
		return MakeBasicAuthenticator(&authConfig.BasicAuthConfig, webSocketTokens), nil
	case authConfig.AuthType == "authorization_code":
		return MakeRsaJwtAuthenticator(&authConfig.AuthorizationCodeConfig, webSocketTokens)
	default:
		return nil, fmt.Errorf("invalid auth_type got: %s expected one of 'basic', 'authorization_code'", authConfig.AuthType)
	}
}

func closeUnauthorized(c *websocket.Conn) {
	deadline := time.Now().Add(time.Second)
	fmt.Println("unauthorized")
	message := websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Unauthorized")
	_ = c.WriteControl(websocket.CloseMessage, message, deadline)
	c.Close()
}

func closeAndHandleErrors(c *websocket.Conn, err error) {
	if err != nil {
		deadline := time.Now().Add(time.Second)
		log.Printf("server error: %v", err)
		message := websocket.FormatCloseMessage(websocket.CloseInternalServerErr, fmt.Sprintf("%v", err))
		_ = c.WriteControl(websocket.CloseMessage, message, deadline)
		c.Close()
		return
	}
	deadline := time.Now().Add(time.Second)
	message := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
	_ = c.WriteControl(websocket.CloseMessage, message, deadline)
	c.Close()
}
