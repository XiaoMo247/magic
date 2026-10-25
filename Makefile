.PHONY: dev backend frontend test build

backend:
	go run ./backend

frontend:
	cd frontend && npm run dev

dev:
	@echo "Open two terminals and run: make backend / make frontend"

test:
	go test ./backend

build:
	go test ./backend
	cd frontend && npm ci && npm run build
