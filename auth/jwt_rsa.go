package auth

import (
	"crypto/rsa"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/optionfactory/journal-webd/pem"
)

type RsaJwtAuthenticator struct {
	PublicKey       *rsa.PublicKey
	WebSocketTokens []string
}

func MakeRsaJwtAuthenticator(config *AuthorizationCodeConfig, webSocketTokens []string) (*RsaJwtAuthenticator, error) {
	key, err := jwt.ParseRSAPublicKeyFromPEM(pem.Armored(pem.PUBLIC_KEY, config.PublicKey))
	if err != nil {
		return nil, err
	}
	return &RsaJwtAuthenticator{
		PublicKey:       key,
		WebSocketTokens: webSocketTokens,
	}, nil
}

func (self *RsaJwtAuthenticator) InterceptAssetRequest(cb func(*fiber.Ctx) error) func(*fiber.Ctx) error {
	return cb
}

func (self *RsaJwtAuthenticator) InterceptApiCall(c *fiber.Ctx) error {
	authorization := c.Get("Authorization")
	if !strings.HasPrefix(authorization, "Bearer ") {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	accessToken := strings.TrimPrefix(authorization, "Bearer ")
	if accessToken == "" {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	parsedToken, err := jwt.Parse(accessToken, func(t *jwt.Token) (interface{}, error) {
		_, ok := t.Method.(*jwt.SigningMethodRSA)
		if !ok {
			return nil, fmt.Errorf("unexpected method: %s", t.Header["alg"])
		}
		return self.PublicKey, nil
	})

	if err != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return c.SendStatus(fiber.StatusForbidden)
	}
	//TODO: process claims
	_ = claims
	return c.Next()
}

func (self *RsaJwtAuthenticator) MakeAuthenticatedWebSocket(cb func(c *websocket.Conn)) func(*fiber.Ctx) error {
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
			fmt.Println("found token")
			token := strings.TrimPrefix(authorization, "Bearer ")
			if slices.Contains(self.WebSocketTokens, token) {
				fmt.Println("token matches")
				cb(c)
				return
			}
		}

		auth := &WebsocketAuthorizationRequest{}
		err := c.ReadJSON(auth)
		if err != nil {
			return
		}
		accessToken := auth.Authorization
		if accessToken == "" {
			return
		}
		parsedToken, err := jwt.Parse(accessToken, func(t *jwt.Token) (interface{}, error) {
			_, ok := t.Method.(*jwt.SigningMethodRSA)
			if !ok {
				return nil, fmt.Errorf("unexpected method: %s", t.Header["alg"])
			}
			return self.PublicKey, nil
		})

		if err != nil {
			return
		}

		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok || !parsedToken.Valid {
			return
		}
		//TODO: process claims
		_ = claims
		cb(c)
	}, websocket.Config{
		WriteBufferSize: 8192,
	})

}
