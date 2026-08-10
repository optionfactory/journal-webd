REPO_OWNER=optionfactory
REPO_NAME=journal-webd
VERSION=v1.2-dev


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


publish-github: build
	gh release create "$(VERSION)" \
		"bin/$(REPO_NAME)-linux-amd64#$(REPO_NAME)-linux-amd64" \
		--repo "$(REPO_OWNER)/$(REPO_NAME)" \
		--title "$(VERSION)" \
		--target "master" \
		--notes "release $(VERSION)"