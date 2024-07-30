package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	queue "github.com/arjun-1/task-queue"
	"github.com/arjun-1/task-queue/internal/db"
)

func main() {
	ctx := context.Background()
	connStr := "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable"

	pgxPool, err := queue.NewDBPool(ctx, connStr)
	if err != nil {
		panic(err)
	}

	if err := queue.Migrate(connStr); err != nil {
		panic(err)
	}

	queries := db.New(pgxPool)

	q := queue.NewQueue("my-queue", queries, 1*time.Second)

	queue.AddHandler(
		"my-kind",
		&q,
		queue.TaskHandler[string]{
			Handler: func(ctx context.Context, t queue.Task[string]) error {
				if t.Priority == 0 {
					time.Sleep(5 * time.Second)
				} else {
					time.Sleep(1 * time.Second)
				}

				return nil
			},
		},
	)

	for i := range 2 {
		payload, err := json.Marshal(fmt.Sprintf("\"hello world!: %d\"", i))
		if err != nil {
			panic(err)
		}

		task, err := queries.TaskInsert(
			ctx,
			db.TaskInsertParams{Payload: payload, Kind: "my-kind", Queue: "my-queue", Priority: int32(-i)},
		)
		if err != nil {
			panic(err)
		}

		fmt.Printf("inserting task (ID: %d)\n", task.ID)
	}

	if err := q.Start(ctx); err != nil {
		panic(err)
	}
}
