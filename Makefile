# Image URL to use all building/pushing image targets
include .settings

# Build the container image
.PHONY: build
build:
	docker build --platform linux/amd64 -t ${URL}${NAME}:v${VERSION} -t ${URL}${NAME}:latest .

# Push the container image
.PHONY: push
push:
	aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws
	docker push ${URL}${NAME}:v${VERSION}
	docker push ${URL}${NAME}:latest

# Default target
.DEFAULT_GOAL := build
