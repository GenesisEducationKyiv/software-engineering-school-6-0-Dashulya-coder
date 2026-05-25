.PHONY: test-integration

test-integration:
	docker compose -f tests/integration/docker-compose.test.yml up -d --wait
	DATABASE_URL="postgres://postgres:postgres@localhost:5433/notifier_test?sslmode=disable" \
	INTEGRATION=1 \
	go test ./tests/integration/... -v -count=1 -timeout=120s; \
	EXIT=$$?; \
	docker compose -f tests/integration/docker-compose.test.yml down -v; \
	exit $$EXIT
