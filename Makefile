# One command for "is this change healthy": run `make check` before pushing; CI runs the same.
GO_PKGS := ./...
WEB := desktop/frontend
WAILS ?= $(shell command -v wails || echo $(HOME)/go/bin/wails)

.PHONY: check fmt vet test web vuln build app bindings app-release

check: fmt vet test web vuln

fmt:
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

vet:
	go vet $(GO_PKGS)

test:
	go test -race $(GO_PKGS)

# Front end: types, unit tests, known vulnerabilities. Needs `npm ci` in $(WEB) once.
web:
	cd $(WEB) && npx tsc --noEmit && npm test --silent && npm audit --audit-level=moderate

# Known vulnerabilities in Go dependencies and the standard library, when govulncheck is installed
# (go install golang.org/x/vuln/cmd/govulncheck@latest).
vuln:
	@if command -v govulncheck >/dev/null; then govulncheck $(GO_PKGS); else echo "skipping govulncheck (not installed)"; fi

build:
	go build -o gh-prs .

app:
	cd desktop && $(WAILS) build -clean=false

# Regenerate the front end's Go bindings. Wails also rewrites its runtime files; keep ours.
bindings:
	cd desktop && $(WAILS) generate module && git checkout -- frontend/wailsjs/runtime

# Attach the macOS app to a release CI made (it needs a Mac to build): make app-release TAG=v0.2.0
app-release: app
	@test -n "$(TAG)" || (echo "usage: make app-release TAG=vX.Y.Z"; exit 1)
	cd desktop/build/bin && rm -f gh-prs-macos.zip && ditto -c -k --keepParent gh-prs.app gh-prs-macos.zip
	gh release upload $(TAG) desktop/build/bin/gh-prs-macos.zip --clobber
