
.PHONY: test
test:
	docker compose run --rm app go test -race ./...
