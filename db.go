package queue

import (
	"context"
	"embed"

	"log/slog"
	"net/url"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"

	pgxslog "github.com/mcosta74/pgx-slog"
)

//go:embed internal/db/migration
var migrations embed.FS

func Migrate(connStr string) error {
	u, err := url.Parse(connStr)
	if err != nil {
		return err
	}

	dbmateDB := dbmate.New(u)
	dbmateDB.FS = migrations
	dbmateDB.MigrationsDir = []string{"internal/db/migration"}
	dbmateDB.AutoDumpSchema = false

	return dbmateDB.CreateAndMigrate()
}

func NewDBPool(ctx context.Context, connStr string) (*pgxpool.Pool, error) {
	pgxConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}

	pgxConfig.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger: pgxslog.NewLogger(slog.Default()), LogLevel: tracelog.LogLevelWarn}

	return pgxpool.NewWithConfig(ctx, pgxConfig)

}
