env-setup:
	curl -sSf https://atlasgo.sh | sh

binary-build:
	go build -o iam ./cmd/iam
	strip iam

image-microservice-build: version ?= latest
image-microservice-build:
	docker buildx build \
		--platform linux/amd64 \
		--file Dockerfile.microservice \
		--tag gdemojkod/stormic-iam-microservice:${version} \
		.

image-microservice-push: version ?= latest
image-microservice-push:
	docker push gdemojkod/stormic-iam-microservice:${version}

image-migration-build: version ?= latest
image-migration-build:
	docker buildx build \
		--platform linux/amd64 \
		--file Dockerfile.migration \
		--tag gdemojkod/stormic-iam-migration:${version} \
		.

image-migration-push: version ?= latest
image-migration-push:
	docker push gdemojkod/stormic-iam-migration:${version}

chart-build: version ?= 0.1.0
chart-build:
	helm package chart --version ${version}

chart-install: version ?= 0.1.0
chart-install:
	helm -n iam install --create-namespace iam iam-${version}.tgz

ent-add-orm: name ?= User
ent-add-orm:
	go run -mod=mod entgo.io/ent/cmd/ent new ${name}

ent-generate:
	go generate ./ent

infrastructure-up:
	docker-compose up -d

infrastructure-down:
	docker-compose down

database-create-migration: name ?= initial
database-create-migration:
	atlas migrate diff ${name} \
		--dir "file://migrations" \
		--to "ent://internal/ent/schema" \
		--dev-url "postgres://postgres:@127.0.0.1:5432/postgres?sslmode=disable"

database-apply-migrations:
	atlas migrate apply \
		--dir "file://migrations" \
		--url "postgres://postgres:@127.0.0.1:5432/postgres?sslmode=disable" --dry-run

graphql-init:
	go get github.com/99designs/gqlgen@latest
	go run github.com/99designs/gqlgen init

graphql-generate:
	go get github.com/99designs/gqlgen@latest
	go generate ./...
