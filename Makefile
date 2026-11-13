# One command for "is this change healthy": run `make check` before pushing; CI runs the same.
GO_PKGS := ./...
WEB := desktop/frontend
WAILS ?= $(shell command -v wails || echo $(HOME)/go/bin/wails)
# Wails runs the app to generate bindings, and the app loads a config: give it an empty one, so
# your own ~/.config/gh-prs can't break a build.
WAILS_RUN = XDG_CONFIG_HOME=$$(mktemp -d) $(WAILS)

.PHONY: check fmt vet test web vuln build app bindings app-release hooks release

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

# Known vulnerabilities in Go dependencies and the standard library (pinned; nothing to install).
vuln:
	go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 $(GO_PKGS)

build:
	go build -o gh-prs .

app:
	cd desktop && $(WAILS_RUN) build -clean=false

# Regenerate the front end's Go bindings. Wails also rewrites its runtime files; keep ours.
bindings:
	cd desktop && $(WAILS_RUN) generate module && git checkout -- frontend/wailsjs/runtime

# Attach the macOS app to a release (it needs a Mac to build): make app-release TAG=v0.2.0
# Universal, so it runs on Intel and Apple silicon Macs.
app-release:
	@test -n "$(TAG)" || (echo "usage: make app-release TAG=vX.Y.Z"; exit 1)
	cd desktop && $(WAILS_RUN) build -clean=false -platform darwin/universal
	cd desktop/build/bin && rm -f gh-prs-macos.zip && ditto -c -k --keepParent gh-prs.app gh-prs-macos.zip
	gh release upload $(TAG) desktop/build/bin/gh-prs-macos.zip --clobber

# Run `make check` before every push from this clone.
hooks:
	git config core.hooksPath scripts/hooks

# Tag a merged PR and publish its release: make release PR=12 (RELEASE_DRY_RUN=1 to rehearse).
release:
	@test -n "$(PR)" || (echo "usage: make release PR=<merged PR number>"; exit 1)
	scripts/release.sh $(PR)
