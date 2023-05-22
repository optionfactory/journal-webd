package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	recovermw "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html"
	"github.com/gofiber/websocket/v2"
	"github.com/optionfactory/journal-webd/auth"
	"github.com/optionfactory/journal-webd/journal"
	"github.com/optionfactory/journal-webd/pem"
)

var version string

type ListenerConfig struct {
	Protocol       string `json:"protocol"` //http, https
	Address        string `json:"address"`
	TlsCertificate string `json:"certificate"`
	TlsKey         string `json:"key"`
}

type Configuration struct {
	JournalsDirectory string                       `json:"journals_directory"`
	ProxyMode         string                       `json:"proxy_mode"`
	UiAuthConfig      *auth.UiAuthConfig           `json:"ui"`
	AllowedHosts      []string                     `json:"allowed_hosts"`
	AllowedUnits      []string                     `json:"allowed_units"`
	WebSocketTokens   []string                     `json:"web_socket_tokens"`
	JournalRemote     *journal.JournalRemoteConfig `json:"journal_remote"`
	Listener          ListenerConfig               `json:"listener"`
}

func LoadConfiguration(filename string) (*Configuration, error) {
	self := &Configuration{}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(self)
	if err != nil {
		return nil, err
	}
	return self, nil
}

type ConfResponse struct {
	Units []string `json:"units"`
	Hosts []string `json:"hosts"`
}

func MakeListener(conf *ListenerConfig) (net.Listener, error) {
	if conf.Protocol == "http" {
		return net.Listen("tcp", conf.Address)
	}
	cer, err := tls.X509KeyPair(pem.Armored(pem.CERTIFICATE, conf.TlsCertificate), pem.Armored(pem.PRIVATE_KEY, conf.TlsKey))
	if err != nil {
		return nil, err
	}
	return tls.Listen("tcp", conf.Address, &tls.Config{
		Certificates: []tls.Certificate{cer},
	})
}

// journal-remote supports log rotation since version 253
func main() {

	if len(os.Args) != 2 {
		log.Fatal(fmt.Sprintf("usage: %s <path-to-configuration.json>", os.Args[0]))
	}
	conf, err := LoadConfiguration(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("starting %s, version: %s", os.Args[0], version)
	authenticator, err := auth.MakeAuthenticator(conf.UiAuthConfig, conf.WebSocketTokens)
	if err != nil {
		log.Fatal(err)
	}

	templateEngine := html.NewFileSystem(http.FS(assets), ".html")
	templateEngine.Delims("@{{", "}}")

	proxyHeader := ""
	if conf.ProxyMode == "edge" {
		proxyHeader = fiber.HeaderXForwardedFor
	}

	gofiber := fiber.New(fiber.Config{
		Views:                 templateEngine,
		Prefork:               false,
		DisableStartupMessage: true,
		ProxyHeader:           proxyHeader,
	})

	gofiber.Use(recovermw.New())
	gofiber.Use(logger.New())

	journalReader := journal.MakeReader(conf.JournalsDirectory, conf.AllowedUnits, conf.AllowedHosts)
	journalRemote := journal.MakeRemote(conf.JournalsDirectory, conf.JournalRemote)

	doneChannel := make(chan bool)

	err = journalRemote.Start(doneChannel)
	if err != nil {
		panic(err)
	}
	if conf.UiAuthConfig != nil {
		gofiber.Get("/", authenticator.InterceptAssetRequest(func(c *fiber.Ctx) error {
			return c.Render("assets/index", conf.UiAuthConfig)
		}))

		gofiber.Get("/app.js", authenticator.InterceptAssetRequest(func(c *fiber.Ctx) error {
			data, _ := assets.Open("assets/app.js")
			return c.SendStream(data)
		}))

		gofiber.Get("/auth.js", authenticator.InterceptAssetRequest(func(c *fiber.Ctx) error {
			data, _ := assets.Open(fmt.Sprintf("assets/auth-%s.js", conf.UiAuthConfig.AuthType))
			return c.SendStream(data)
		}))

		gofiber.Use("/api", authenticator.InterceptApiCall)

		allowedHosts, err := journalReader.KnownAllowedHosts()
		if err != nil {
			log.Fatal(err)
		}
		allowedUnits, err := journalReader.KnownAllowedUnits()
		if err != nil {
			log.Fatal(err)
		}

		gofiber.Get("/api/conf", func(c *fiber.Ctx) error {
			return c.JSON(&ConfResponse{
				Hosts: allowedHosts,
				Units: allowedUnits,
			})
		})
	}
	gofiber.Get("/ws/stream", authenticator.MakeAuthenticatedWebSocket(func(c *websocket.Conn) {
		req := &journal.StreamRequest{}
		c.ReadJSON(req)
		journalReader.Stream(c, req)
	}))
	log.Printf("listening...")
	ln, err := MakeListener(&conf.Listener)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		e := gofiber.Listener(ln)
		log.Printf("gofiber quit: err? %v", e)
		doneChannel <- true
	}()
	<-doneChannel
	log.Printf("done")
}
