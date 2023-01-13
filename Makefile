# The binary to build (just the basename).
BIN := grafana-db-migrator
SRC_DIR := ./cmd/grafana-migrate

# This version-strategy uses git tags to set the version string
VERSION := $(shell git describe --tags --always --dirty)
OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)
TAG := $(shell git tag | tail -1)
PWD := $(shell pwd)


build:
	go build -o dist/$(BIN) $(SRC_DIR)

build-all:
	env GOOS=darwin GOARCH=amd64 go build -o dist/$(BIN)_darwin_amd64-$(VERSION) $(SRC_DIR)
	env GOOS=linux GOARCH=amd64 go build -o dist/$(BIN)_linux_amd64-$(VERSION) $(SRC_DIR)
	env GOOS=windows GOARCH=amd64 go build -o dist/$(BIN)_windows_amd64-$(VERSION).exe $(SRC_DIR)

# For manually releasing when Drone Cloud is having issues
manual-release: build-all
	docker run --rm \
		-e DRONE_BUILD_EVENT=tag \
		-e DRONE_REPO_OWNER=wbh1 \
		-e DRONE_REPO_NAME='grafana-sqlite-to-postgres' \
		-e DRONE_COMMIT_REF="refs/tags/$(TAG)" \
		-e PLUGIN_TITLE="$(TAG)" \
		-e PLUGIN_API_KEY="${GITHUB_TOKEN}" \
		-e PLUGIN_FILES='dist/*' \
		-e PLUGIN_OVERWRITE='true' \
		-e DRONE_REPO_LINK='https://github.com/wbh1/grafana-sqlite-to-postgres' \
		-v "$(PWD):$(PWD)" \
		-w "$(PWD)" \
		plugins/github-release

clean:
	rm dist/*

env-run:
	env GOOS=linux GOARCH=amd64 go build -o dist/$(BIN) $(SRC_DIR)
	docker cp dist/$(BIN) pmm-server:/
	docker exec -t pmm-server /grafana-db-migrator --change-char-to-text --reset-home-dashboard /srv/backup/grafana/grafana.db "postgres://grafana:grafana@localhost:5432/grafana?sslmode=disable"
