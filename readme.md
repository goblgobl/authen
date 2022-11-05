# Authentication Enhancement
A service for adding 2FA and tickets to an existing authentication flow.  See the [documentation](https://www.goblgobl.com/docs/authen) for more information on what it does and how to use it.

## Development
Requires Go (1.18+), PostgreSQL and CockroachDB. Use `make t` to run all tests. We don't use Docker because its startup/teardown time is noticeable enough to be annoying when quickly iterating (though of course, you can use what you want). 

You can set the `GOBL_TEST_PG` and `GOBL_TEST_CR` environment variables to the full postgres and cockroach connection URLs, they default to: `postgres://localhost:5432` and `postgres://root@localhost:26257` respectively. A `gobl_test` database will automatically be created.
