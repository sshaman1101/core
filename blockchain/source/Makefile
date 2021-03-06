#!/usr/bin/env make
ifeq ($(GO), )
    GO=go
endif
TRUFFLE=./node_modules/truffle/build/cli.bundled.js
TESTRPC=./node_modules/ethereumjs-testrpc/build/cli.node.js
SOLIUM=./node_modules/solium/bin/solium.js
ESLINT=./node_modules/.bin/eslint

.PHONY: all test clean node_modules coverage api

all: node_modules lint compile test

compile: clean
	@echo "+ $@"
	@${TRUFFLE} compile

node_modules:
	@echo "+ $@"
	@npm install

test:
	@echo "+ $@"
	@scripts/test.sh

lint: lint_sol lint_js

lint_sol:
	@echo "+ $@"
	@${SOLIUM} --dir ./contracts

lint_js:
	@echo "+ $@"
	@${ESLINT} .

migrate:
	${TRUFFLE} migrate

coverage:
	scripts/test_coverage.sh

deploy:
	${TRUFFLE} migrate --network private

clean:
	@rm -rf build

generate_api: clean compile
	${GO} build utils/generate_api.go
	./generate_api
	rm generate_api
