.PHONY: tests
tests:
	@echo "--- Unitary tests ---"
	go test ./... -run=^Test_U_ -json -cover -coverprofile=cov.out | tee -a gotest.json
