package queue

import (
	"context"
	"time"

	"github.com/arjun-1/task-queue/internal/db"
)

// TaskHandler configures the generic handling of a Task[T]
type TaskHandler[T any] struct {
	Handler func(context.Context, Task[T]) error
	Retry   RetryConfig
}

type RetryConfig struct {
	Delay       func(task db.Task, err error) time.Duration
	MaxAttempts int
}

type Task[T any] struct {
	db.Task
	TypedPayload T
}
