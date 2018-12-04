BINARY=xdiff

VERSION=`cat ./VERSION`
BUILD=`date +%FT%T%z`
BUILD_DIR=build/$(VERSION)

LDFLAGS=-ldflags "-w -s -X main.version=$(VERSION) -X main.date=$(BUILD)"

all: install

install:
	@echo "Installing xdiff version $(VERSION)."
	@go install -i $(LDFLAGS) ./cmd/xdiff

test:
	@echo "Testing xdiff version $(VERSION)."
	@go test -v -race . ./xtree ./parser

build: fmt
	@echo "Building version $(VERSION)."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/xdiff ./cmd/xdiff/*.go

fmt:
	@go fmt ./...

release: confirmation
	@git tag -a $(VERSION) -m "Release $(VERSION)" || true
	@git push origin $(VERSION)
	@goreleaser --rm-dist

confirmation:
	@echo "XDiff version: $(VERSION)"
	@( read -p "Are you sure?!? [y/N]: " sure && case "$$sure" in [yY]) true;; *) false;; esac )

.PHONY: confirmation install test fmt release