# task-queue

This project implements an asynchronous task queue backed by Postgres. The idea is not new as projects like [River Queue](https://riverqueue.com/) and [Hatchet](https://hatchet.run/) have recently done the same.

They all exploit a recently new feature added to Postgres since 9.5: `SKIP LOCKED`. In combination with explicit locking `FOR UPDATE`. Essentially it allows attempting to acquire a lock on rows to be dequeued, and ignore any elements if it can't acquire any. This prevents issues that would have occured with transaction isolation: if 2 consumers would have dequeued at the same time, leading to 'lost update' problem.

We use the same clause.

## Usage

After connecting and migrating the DB, a `queue` is instantiated via

```golang
q := queue.NewQueue("my-queue", queries, 1*time.Second)
```

which configures a queue which polls every second from the table.

A handler should then be attached, after which the queue can be started (initiating the polling of tasks):

```golang
queue.AddHandler(
  "string",
  &q,
  queue.TaskHandler[string]{
    Handler: func(ctx context.Context, t queue.Task[string]) error {
      fmt.Printf("done: %s\n", t.TypedPayload)

      return nil
    },
  },
)

q.Start(ctx)
```

To enqueue an element, simply insert to the DB:

```golang
_, err := queries.TaskInsert(
  ctx,
  db.TaskInsertParams{Payload: payload, Kind: "string", Queue: "my-queue"},
)
```

## Architecture

We use Postgres table as an event queue, allowing multiple consumers to consume from the same queue. In principle an event an only be consumed by a single consumer, but it strives to deliver at least once.

The reason is that we chose to model a task to have a state (`available`, `failed`, `completed` or `running`), instead of keeping an offset in the queue per consumer to keep things simple.

The database model and queries are written in raw SQL (see `internal/db`) and compiled to Go code using [sqlc](https://sqlc.dev/).

### Fetching

There is a goroutine (`fetchTaskLoop`) which continously polls the table for available tasks. It does so with backpressure, taking into account the specified max number of workers or buffer size. The fetched tasks are send to a channel to be handled by the worker pool.

### Handling

The fetched tasks are read from the channel in different goroutines (`handleTaskLoop`). The number of goroutines working in parallel is limited via a 'semaphore' pattern.

During handling a task can:

- succeed without errors, in which case it becomes completed.
- fail and be eligible for retry, in which case it is retried. Any backoff is respected by `fetchTaskLoop`: a retried task is reinserted back into the table as available, but having a corresponding `scheduledAt` value.
- fail and not be eligible for retry (if the maximum attempts have occurred), in which case it becomes failed.

## DEV

The application has 3 main entrypoints: `example_many`, `example_priority` and `example_retry` showcasing different features.

The first example can be started via Docker:

```bash
docker-compose up
```

or manually (requiring a Postgres DB):

```bash
go run cmd/example_many/main.go
```

## Improvements

Due to time restrictions, some shortcuts were taken leaving room for the following improvements:

- In theory a task can become 'lost' if the application crashes leaving it permantly in a running state without it being worked on. This can be addressed by a rescue process, carried out by an elected leader among clients.
- Not everywhere do we respect context cancellation.
- Much of the configuration of a `Queue` is now hardcoded.
- There is no API for inserting a task.
- All the logic still resides in one package.
- Only some part of the DB queries are tested leaving most of the domain logic
  uncovered.
- Testing spawns a postgres DB which is not reused between tests.
