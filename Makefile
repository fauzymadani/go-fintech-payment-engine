SHELL := /bin/bash

.PHONY: help proto up down seed run

help:
	@echo "Makefile commands:"
	@echo "  make proto   - generate protobufs with buf"
	@echo "  make up      - start services with podman-compose"
	@echo "  make down    - stop services with podman-compose"
	@echo "  make seed    - run the DB seeder"
	@echo "  make run     - run the transaction service"

proto:
	buf generate

up:
	podman-compose up -d

down:
	podman-compose down

seed:
	go run cmd/seed/main.go

run:
	go run services/transaction/main.go

