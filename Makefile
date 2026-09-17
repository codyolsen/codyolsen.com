PORT ?= 8000

.PHONY: run
run:
	@echo "Game: http://localhost:$(PORT)/game/ (Ctrl+C to stop)"
	go run ./cmd/serve -port $(PORT)
