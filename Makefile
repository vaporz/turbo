.PHONY: install
install:
	@cd turbo && go install
	@cd protoc-gen-buildfields && go install

.PHONY: test
test:
	@go test -cover -coverpkg github.com/vaporz/turbo github.com/vaporz/turbo github.com/vaporz/turbo/test
	@cd test/testcreateservice && go build ./...
	@cd test/testservice && go build ./...

.PHONY: doc
doc:
	@cd doc && make html
