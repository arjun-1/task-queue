package db_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/arjun-1/task-queue"
	"github.com/arjun-1/task-queue/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestTaskListAvailable(t *testing.T) {
	ctx := context.Background()

	pgxConn, c := setupDB(ctx, t)
	defer c.Terminate(ctx)

	queries := db.New(pgxConn)

	payload, err := json.Marshal("\"hello world!\"")
	require.NoError(t, err)

	insertedTask, err := queries.TaskInsert(
		ctx,
		db.TaskInsertParams{Payload: payload, Kind: "string", Queue: "queue"},
	)

	availableTasks, err := queries.TaskListAvailable(ctx, db.TaskListAvailableParams{
		Queue: "queue",
		Max:   5,
	})

	require.NoError(t, err)

	assert.Len(t, availableTasks, 1)
	assert.Equal(t, insertedTask.ID, availableTasks[0].ID)
}

func TestTaskComplete(t *testing.T) {
	ctx := context.Background()

	pgxConn, c := setupDB(ctx, t)
	defer c.Terminate(ctx)

	queries := db.New(pgxConn)

	payload, err := json.Marshal("\"hello world!\"")
	require.NoError(t, err)

	insertedTask, err := queries.TaskInsert(
		ctx,
		db.TaskInsertParams{Payload: payload, Kind: "string", Queue: "queue"},
	)
	require.NoError(t, err)

	_, err = queries.TaskListAvailable(ctx, db.TaskListAvailableParams{Queue: "queue", Max: 5})
	require.NoError(t, err)

	_, err = queries.TaskComplete(ctx, insertedTask.ID)

	availableTasks, err := queries.TaskListAvailable(ctx, db.TaskListAvailableParams{
		Queue: "queue",
		Max:   5,
	})
	require.NoError(t, err)

	assert.Len(t, availableTasks, 0)
}

func TestTaskRetry(t *testing.T) {
	ctx := context.Background()

	pgxConn, _ := setupDB(ctx, t)

	queries := db.New(pgxConn)

	payload, err := json.Marshal("\"hello world!\"")
	require.NoError(t, err)

	insertedTask, err := queries.TaskInsert(
		ctx,
		db.TaskInsertParams{Payload: payload, Kind: "string", Queue: "queue"},
	)
	require.NoError(t, err)

	_, err = queries.TaskListAvailable(ctx, db.TaskListAvailableParams{Queue: "queue", Max: 5})
	require.NoError(t, err)

	_, err = queries.TaskRetry(ctx, db.TaskRetryParams{
		ID:          insertedTask.ID,
		Err:         "oops",
		Scheduledat: pgtype.Timestamptz{Valid: true, Time: time.Now().Add(-time.Second)},
	})
	require.NoError(t, err)

	availableTasks, err := queries.TaskListAvailable(ctx, db.TaskListAvailableParams{
		Queue: "queue",
		Max:   5,
	})
	require.NoError(t, err)

	assert.Len(t, availableTasks, 1)
	assert.Equal(t, "oops", availableTasks[0].Error.String)
}

func setupDB(ctx context.Context, t *testing.T) (*pgx.Conn, *postgres.PostgresContainer) {
	container, err := postgres.Run(
		ctx,
		"docker.io/postgres:16-alpine",
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to run postgres container: %s", err)
	}

	port, _ := container.MappedPort(ctx, "5432/tcp")
	host, _ := container.Host(ctx)

	connStr := fmt.Sprintf("postgresql://postgres:postgres@%s:%s/postgres?sslmode=disable", host, port)

	err = queue.Migrate(connStr)
	if err != nil {
		t.Fatalf("failed to migrate DB: %s", err)
	}

	pool, err := pgx.Connect(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to connect to DB: %s", err)
	}

	return pool, container
}

func resetDB(ctx context.Context, conn *pgxpool.Pool) error {
	_, err := conn.Exec(ctx, "truncate task")
	return err
}
