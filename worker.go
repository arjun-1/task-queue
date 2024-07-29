package queue

import (
	"context"
	"encoding/json"

	"github.com/arjun-1/task-queue/internal/db"
)

// Ensure that TaskWorker[T] implements the Worker interface.
var _ Worker = TaskWorker[any]{}

// TaskWorker is a worker that is scoped to perform exactly one task.
type TaskWorker[T any] struct {
	handler TaskHandler[T]
	task    *Task[T]
}

func (w TaskWorker[T]) UnmarshalPayload() error {
	return json.Unmarshal(w.task.Payload, &w.task.TypedPayload)
}

func (w TaskWorker[T]) Handle(ctx context.Context) error {
	return w.handler.Handler(ctx, *w.task)
}

func (w TaskWorker[T]) RetryConfig() RetryConfig {
	return w.handler.Retry
}

func MakeWorker[T any](handler TaskHandler[T]) func(dbTask db.Task) Worker {
	return func(dbTask db.Task) Worker {
		return TaskWorker[T]{
			handler,
			&Task[T]{Task: dbTask},
		}
	}
}
