.PHONY: test-integration test-e2e

test-integration:
	docker compose -f tests/integration/docker-compose.test.yml up -d --wait
	DATABASE_URL="postgres://postgres:postgres@localhost:5433/notifier_test?sslmode=disable" \
	INTEGRATION=1 \
	go test ./tests/integration/... -v -count=1 -timeout=120s; \
	EXIT=$$?; \
	docker compose -f tests/integration/docker-compose.test.yml down -v; \
	exit $$EXIT

test-e2e:
	docker compose -f tests/e2e/docker-compose.e2e.yml up -d --build --wait
	E2E_BASE_URL=http://localhost:8080 npx playwright test; \
	EXIT=$$?; \
	docker compose -f tests/e2e/docker-compose.e2e.yml down -v; \
	exit $$EXIT
