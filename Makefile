F=

.PHONY: t
t: commit.txt
	@printf "running tests against sqlite\n"
	@GOBL_TEST_STORAGE=sqlite go test -count=1 ./... -run "${F}" \
		| grep -v "no tests to run" \
		| grep -v "no test files"

	@printf "\nrunning tests against postgres\n"
	@GOBL_TEST_STORAGE=pg go test -count=1 ./... -run "${F}" \
		| grep -v "no tests to run" \
		| grep -v "no test files"

.PHONY: s
s: commit.txt
	go run cmd/main.go
	go run cmd/main.go

.PHONY: commit.txt
commit.txt:
	@git rev-parse HEAD | tr -d "\n" > http/misc/commit.txt
