ENV ?= dev

ENV_FILE   := .env.$(ENV)
ENV_SAMPLE := .env.$(ENV).sample

DOCKER_COMPOSE := docker compose
COMPOSE_FILE   := docker-compose.yml

PROJECT_NAME := solidarytech

.DEFAULT_GOAL := help

.PHONY: \
	help \
	check \
	check-all \
	docker-up \
	docker-down \
	docker-ps \
	kaboom \
	localstack-setup

ifeq ($(wildcard $(ENV_SAMPLE)),)
  $(error Missing environment sample file: $(ENV_SAMPLE))
endif

ifeq ($(wildcard $(ENV_FILE)),)
  $(info Creating environment file from $(ENV_SAMPLE))
  $(shell cp $(ENV_SAMPLE) $(ENV_FILE))
endif

-include $(ENV_FILE)
export

REQUIRES_ENV := \
	docker-up \
	docker-down \
	docker-ps \
	kaboom \
	check \
	check-all

ifneq ($(filter $(REQUIRES_ENV),$(MAKECMDGOALS)),)

  REQUIRED_VARS := $(shell \
    grep -E '^[A-Z0-9_]+\??=' $(ENV_SAMPLE) | \
    grep -v '\?=' | \
    cut -d '=' -f1 \
  )

  $(foreach var,$(REQUIRED_VARS),\
    $(if $(value $(var)),,\
      $(error Missing required variable '$(var)' in $(ENV_FILE) (ENV=$(ENV)))\
    )\
  )

endif

help:
	@printf "\nUsage: make <target> [ENV=dev|prod]\n\n"
	@printf "Targets:\n"
	@printf "  %-16s %s\n" "check <svc>"      "Run service check"
	@printf "  %-16s %s\n" "check-all"        "Run all checks"
	@printf "  %-16s %s\n" "docker-up"        "Start containers"
	@printf "  %-16s %s\n" "docker-down"      "Stop containers"
	@printf "  %-16s %s\n" "docker-ps"        "List containers"
	@printf "  %-16s %s\n" "kaboom"           "Destroy everything"
	@printf "  %-16s %s\n" "localstack-setup" "Setup LocalStack"
	@printf "\nExamples:\n"
	@printf "  make docker-up ENV=dev\n"
	@printf "  make check-all ENV=prod\n\n"

check:
	@SERVICE="$(word 2,$(MAKECMDGOALS))"; \
	URL="$(word 3,$(MAKECMDGOALS))"; \
	if [ -z "$$SERVICE" ]; then \
		echo "Usage: make check <service> [base_url]"; exit 1; \
	fi; \
	SCRIPT="scripts/check/$$SERVICE.sh"; \
	if [ ! -f "$$SCRIPT" ]; then \
		echo "Check script not found for $$SERVICE"; exit 1; \
	fi; \
	$$SCRIPT $$URL

check-all:
	@./scripts/check/all.sh

docker-up:
	$(DOCKER_COMPOSE) --env-file $(ENV_FILE) -f $(COMPOSE_FILE) up -d --build

docker-down:
	$(DOCKER_COMPOSE) --env-file $(ENV_FILE) -f $(COMPOSE_FILE) down

docker-ps:
	$(DOCKER_COMPOSE) --env-file $(ENV_FILE) -f $(COMPOSE_FILE) ps

kaboom:
	$(DOCKER_COMPOSE) --env-file $(ENV_FILE) -f $(COMPOSE_FILE) down -v --rmi local --remove-orphans

localstack-setup:
	@echo "Creating SQS queue 'solidary-donations'..."
	@docker exec solidarytech-localstack awslocal sqs create-queue --queue-name solidary-donations || echo "Queue exists"
	@echo "Creating DynamoDB table 'SolidaryTechVolunteers'..."
	@docker exec solidarytech-localstack awslocal dynamodb create-table \
		--table-name SolidaryTechVolunteers \
		--attribute-definitions AttributeName=volunteer_id,AttributeType=S \
		--key-schema AttributeName=volunteer_id,KeyType=HASH \
		--provisioned-throughput ReadCapacityUnits=5,WriteCapacityUnits=5 || echo "Table exists"

%:
	@:
