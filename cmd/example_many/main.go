package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	queue "github.com/arjun-1/task-queue"
	"github.com/arjun-1/task-queue/internal/db"
)

func main() {
	ctx := context.Background()

	config, err := queue.AppConfigFromEnv(ctx)
	if err != nil {
		panic(err)
	}

	pgxPool, err := queue.NewDBPool(ctx, config.PostgresURL.String())
	if err != nil {
		panic(err)
	}

	if err := queue.Migrate(config.PostgresURL.String()); err != nil {
		panic(err)
	}

	queries := db.New(pgxPool)

	q := queue.NewQueue("my-queue", queries, 1*time.Second)

	queue.AddHandler(
		"string",
		&q,
		queue.TaskHandler[string]{
			Handler: func(ctx context.Context, t queue.Task[string]) error {
				// sleep anywhere between 3 and 7 seconds
				time.Sleep(time.Duration(rand.Intn(3)+4) * time.Second)

				fmt.Printf("done: %s\n", t.TypedPayload)

				return nil
			},
		},
	)

	for i := range 20 {
		payload, err := json.Marshal(fmt.Sprintf("\"hello world!: %d\"", i))
		if err != nil {
			panic(err)
		}

		task, err := queries.TaskInsert(
			ctx,
			db.TaskInsertParams{Payload: payload, Kind: "string", Queue: "my-queue"},
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
