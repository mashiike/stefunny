
.PHONY: setup
setup:
	docker compose up -d --remove-orphans localstack sfn_local

teardown:
	docker compose down

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
