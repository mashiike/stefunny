
.PHONY: setup
setup:
	docker compose up -d --remove-orphans


.PHONY: plan
plan:
	cd testdata && \
		terraform init -upgrade && \
		terraform plan

.PHONY: apply
apply:
	cd testdata && \
		terraform apply
