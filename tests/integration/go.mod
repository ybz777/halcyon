module github.com/halcyon-io/halcyon/tests/integration

go 1.25.0

require (
	github.com/halcyon-io/halcyon/pkg/platform v0.0.0
	github.com/jackc/pgx/v5 v5.7.4
	github.com/redis/go-redis/v9 v9.7.0
	github.com/stretchr/testify v1.11.1
)

replace github.com/halcyon-io/halcyon/pkg/platform => ../../pkg/platform
