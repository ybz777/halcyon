module github.com/halcyon-io/halcyon/cmd/organization-consumer

go 1.25.0

require github.com/halcyon-io/halcyon/pkg/platform v0.0.0

require (
	github.com/jackc/pgx/v5 v5.7.4
	github.com/segmentio/kafka-go v0.4.47
)

replace github.com/halcyon-io/halcyon/pkg/platform => ../../pkg/platform
