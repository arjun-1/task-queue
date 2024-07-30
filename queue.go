package queue

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/arjun-1/task-queue/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

// Worker defines the non-generic behaviour to perform a task.
// It must be non-generic so that it can perform the handling of a task with
// any `kind`.
// Its generic payload is kept as internal state of the implementation.
type Worker interface {
	UnmarshalPayload() error
	Handle(ctx context.Context) error
	RetryConfig() RetryConfig
}

type Queue struct {
	name            string
	maxWorkers      int
	numWorkers      atomic.Int32
	kindToWorkerGen map[string]func(db.Task) Worker

	fetchBufferSize int
	fetchInterval   time.Duration
	fetchedTaskCh   chan db.Task

	querier db.Querier
}

func NewQueue(name string, querier db.Querier, fetchInterval time.Duration) Queue {
	fetchBufferSize := 10

	return Queue{
		name:            name,
		maxWorkers:      5,
		fetchBufferSize: fetchBufferSize,
		fetchInterval:   fetchInterval,
		kindToWorkerGen: make(map[string]func(db.Task) Worker),
		fetchedTaskCh:   make(chan db.Task, fetchBufferSize),
		querier:         querier,
	}
}

// Start begins the consumption of tasks in the queue. It begins
// two independent loops: the fetching of records and handling of the result.
func (q *Queue) Start(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		if err := q.fetchTaskLoop(ctx); err != nil {
			errCh <- err
		}
	}()

	go func() {
		if err := q.handleTaskLoop(ctx); err != nil {
			errCh <- err
		}
	}()

	for err := range errCh {
		return err
	}

	return nil
}

func AddHandler[T any](kind string, q *Queue, handler TaskHandler[T]) error {
	q.kindToWorkerGen[kind] = MakeWorker(handler)

	return nil
}

func (q *Queue) handleTaskLoop(ctx context.Context) error {
	semaphore := make(chan struct{}, q.maxWorkers)
	errCh := make(chan error, 1)

	go func() {
		for task := range q.fetchedTaskCh {
			semaphore <- struct{}{}

			go func(task db.Task) {
				defer func() { <-semaphore }()

				if err := q.handleTask(ctx, task); err != nil {
					errCh <- err
				}

			}(task)
		}
	}()

	for err := range errCh {
		return err
	}

	return nil
}

func (q *Queue) handleTask(ctx context.Context, task db.Task) error {
	q.numWorkers.Add(1)
	defer q.numWorkers.Add(-1)

	makeWorker := q.kindToWorkerGen[task.Kind]
	worker := makeWorker(task)

	fmt.Printf("spawned worker for task (ID: %d)\n", task.ID)

	retryConfig := worker.RetryConfig()

	if err := worker.UnmarshalPayload(); err != nil {
		fmt.Printf("failed to unmarshal payload: %s\n", err.Error())
		if retryErr := q.retry(ctx, task, retryConfig, err); retryErr != nil {
			return retryErr
		}
		return nil
	}

	if err := worker.Handle(ctx); err != nil {
		fmt.Printf("failed to handle: %s\n", err.Error())
		if retryErr := q.retry(ctx, task, retryConfig, err); retryErr != nil {
			return retryErr
		}
	}

	fmt.Printf("completed task (ID: %d)\n", task.ID)
	if _, err := q.querier.TaskComplete(ctx, task.ID); err != nil {
		return fmt.Errorf("failed to persist task complete: %w", err)
	}
	return nil
}

func (q *Queue) retry(ctx context.Context, task db.Task, retryConfig RetryConfig, err error) error {
	if int(task.Attempt)+1 > retryConfig.MaxAttempts {
		fmt.Printf("failed task (ID: %d)\n", task.ID)
		if _, querierErr := q.querier.TaskFail(ctx, db.TaskFailParams{ID: task.ID, Err: err.Error()}); err != nil {
			return fmt.Errorf("failed to persist task fail: %w", querierErr)
		}
		return nil
	}

	delay := retryConfig.Delay(task, err)
	fmt.Printf("retrying task (ID: %d, error: %s)\n", task.ID, err.Error())
	if _, querierErr := q.querier.TaskRetry(
		ctx,
		db.TaskRetryParams{
			ID:          task.ID,
			Err:         err.Error(),
			Scheduledat: pgtype.Timestamptz{Valid: true, Time: time.Now().Add(delay)},
		},
	); querierErr != nil {
		return fmt.Errorf("failed to persist task retry: %w", querierErr)
	}
	return nil
}

// fetchTaskLoop fetches available tasks with backpressure at fetchInterval.
func (q *Queue) fetchTaskLoop(ctx context.Context) error {
	ticker := time.NewTicker(q.fetchInterval)
	defer ticker.Stop()

	for range ticker.C {
		numWorkers := int(q.numWorkers.Load())
		bufferSize := len(q.fetchedTaskCh)

		limit := max(q.maxWorkers-numWorkers, q.fetchBufferSize-bufferSize)
		fmt.Printf("fetching tasks (workers: %d, buffer: %d, limit: %d)\n", numWorkers, bufferSize, limit)

		tasks, err := q.querier.TaskListAvailable(ctx, db.TaskListAvailableParams{Queue: q.name, Max: int32(limit)})
		if err != nil {
			return err
		}

		fmt.Printf("fetched tasks (num: %d)\n", len(tasks))
		for _, t := range tasks {
			q.fetchedTaskCh <- t
		}
	}

	return nil
}
