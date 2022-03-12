
.PHONY: setup
setup:
	docker compose up -d --remove-orphans localstack sfn_local

.PHONY: teardown
teardown:
	docker compose down

.PHONY: run
run:
	docker compose run --rm app bash

.PHONY: plan
plan:
	cd testdata && \
		terraform init -upgrade && \
		terraform plan

.PHONY: apply
apply:
	cd testdata && \
		terraform apply
