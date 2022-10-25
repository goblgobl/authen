F=

.PHONY: t
t: commit.txt t_sqlite t_pg t_cr

.PHONY: t_sqlite
t_sqlite:
	@printf "\nrunning tests against sqlite\n"
	@GOBL_TEST_STORAGE=sqlite go test -count=1 ./... -run "${F}" \
		| grep -v "no tests to run" \
		| grep -v "no test files"


.PHONY: t_pg
t_pg:
	@printf "\nrunning tests against postgres\n"
	@GOBL_TEST_STORAGE=pg go test -count=1 ./... -run "${F}" \
		| grep -v "no tests to run" \
		| grep -v "no test files"

.PHONY: t_cr
t_cr:
	@printf "\nrunning tests against cockroachdb\n"
	@GOBL_TEST_STORAGE=cr go test -count=1 ./... -run "${F}" \
		| grep -v "no tests to run" \
		| grep -v "no test files"

.PHONY: s
s: commit.txt
	go run cmd/main.go
	go run cmd/main.go

.PHONY: commit.txt
commit.txt:
	@git rev-parse HEAD | tr -d "\n" > http/misc/commit.txt
