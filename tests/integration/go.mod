module github.com/halcyon-io/halcyon/tests/integration

go 1.26.0

require (
	github.com/jackc/pgx/v5 v5.7.4
	github.com/redis/go-redis/v9 v9.23.0
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	go.uber.org/atomic v1.12.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.37.0 // indirect
)

replace github.com/halcyon-io/halcyon/pkg/platform => ../../pkg/platform
