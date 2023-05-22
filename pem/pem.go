package pem

import (
	"fmt"
)

type Type string

const (
	PUBLIC_KEY  Type = "PUBLIC KEY"
	PRIVATE_KEY Type = "PRIVATE KEY"
	CERTIFICATE Type = "CERTIFICATE"
)

func Armored(t Type, content string) []byte {
	armored := fmt.Sprintf("-----BEGIN %s-----\r\n%s\r\n-----END %s-----", t, content, t)
	return []byte(armored)
}
