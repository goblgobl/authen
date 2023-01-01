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
	@psql $${GOBL_TEST_PG:-postgres://localhost:5432/postgres} --quiet -c "drop database gobl_test" || true
	@psql $${GOBL_TEST_PG:-postgres://localhost:5432/postgres} --quiet -c "create database gobl_test"
	@GOBL_TEST_STORAGE=postgres go test -count=1 ./... -run "${F}" \
		| grep -v "no tests to run" \
		| grep -v "no test files"

.PHONY: t_cr
t_cr:
	@printf "\nrunning tests against cockroachdb\n"
	@psql $${GOBL_TEST_CR:-postgres://root@localhost:26257/} --quiet -c "drop database gobl_test" || true
	@psql $${GOBL_TEST_CR:-postgres://root@localhost:26257/} --quiet -c "create database gobl_test"
	@GOBL_TEST_STORAGE=cockroach go test -count=1 ./... -run "${F}" \
		| grep -v "no tests to run" \
		| grep -v "no test files"

.PHONY: s
s: commit.txt
	go run cmd/main.go
	go run cmd/main.go

.PHONY: commit.txt
commit.txt:
	@git rev-parse HEAD | tr -d "\n" > http/misc/commit.txt

.PHONY: build
build: commit.txt
	go build -ldflags="-s -w" -o authen cmd/main.go
