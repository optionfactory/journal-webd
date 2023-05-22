VERSION=$(shell git describe --always --tags --dirty)
REPO_NAME=journal-webd
REPO_OWNER=optionfactory
ARTIFACT_NAME=$(REPO_NAME)-$(VERSION)



$(REPO_NAME): FORCE
	@echo reformatting…
	@find . -name '*.go' | xargs gofmt -w=true -s=true
	@echo testing…
	@CGO_ENABLED=0 go test -a $(TESTING_OPTIONS) -ldflags "-X main.version=$(VERSION)" ./...
	@echo building $(REPO_NAME)…
	@CGO_ENABLED=0 go build -a -ldflags "-X main.version=$(VERSION)"

dev:
	~/go/bin/gow -c -v -e=go -e=js -e=html -e=mod run . local/configuration.json
	#@CGO_ENABLED=0 go build -a -ldflags "-X main.version=$(VERSION)" && ./${REPO_NAME} local/configuration.json

dev-deps:
	go install github.com/mitranim/gow@latest

clean:
	-rm -f qaplastores



FORCE:





release: $(REPO_NAME)
	$(eval github_token=$(shell echo url=https://github.com/$(REPO_OWNER)/$(REPO_NAME) | git credential fill | grep '^password=' | sed 's/password=//'))
	$(eval release_id=$(shell curl -X POST \
		-H "Accept: application/vnd.github+json" \
		-H "Authorization: Bearer $(github_token)" \
		-H "X-GitHub-Api-Version: 2022-11-28" \
		https://api.github.com/repos/$(REPO_OWNER)/$(REPO_NAME)/releases \
	  	-d '{"tag_name":"$(VERSION)","name":"$(VERSION)"}' | jq .id))
	@curl -X POST \
		-H "Accept: application/vnd.github+json" \
		-H "Authorization: Bearer $(github_token)" \
		-H "X-GitHub-Api-Version: 2022-11-28" \
		-H "Content-Type: application/octet-stream" \
		https://uploads.github.com/repos/$(REPO_OWNER)/$(REPO_NAME)/releases/$(release_id)/assets?name=$(ARTIFACT_NAME) \
  		--data-binary "$(REPO_NAME)"

