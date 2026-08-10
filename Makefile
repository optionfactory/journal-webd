REPO_OWNER=optionfactory
REPO_NAME=journal-webd
VERSION=1.2-dev

build: bin/$(REPO_NAME)-linux-amd64

bin/$(REPO_NAME)-linux-amd64: $(SRCS) go.mod
	@mkdir -p bin
	@echo "Formatting and vetting..."
	@go fmt ./...
	@go vet ./...
	@echo "Building $(REPO_NAME)..."
	@CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/$(REPO_NAME)-linux-amd64 .


dev:
	~/go/bin/gow -c -v -e=go -e=js -e=html -e=mod run . local/configuration.json
	#@CGO_ENABLED=0 go build -a -ldflags "-X main.version=$(VERSION)" && ./${REPO_NAME} local/configuration.json

clean:
	@echo "Removing $(REPO_NAME)..."
	@rm -rf bin/

check-updates:
	#go install golang.org/x/vuln/cmd/govulncheck@latest
	-@govulncheck -show verbose  ./...
	#go install github.com/securego/gosec/v2/cmd/gosec@latest
	-@gosec ./...
	@echo Available direct updates
	@go list -u -m -f '{{if and (not .Indirect) .Update}}{{.Path}}: {{.Version}} -> {{.Update.Version}}{{end}}' all


publish-github: build
	gh release create "v$(VERSION)" \
		"bin/$(REPO_NAME)-linux-amd64" \
		--repo "$(REPO_OWNER)/$(REPO_NAME)" \
		--title "v$(VERSION)" \
		--target "master" \
		--notes ""
