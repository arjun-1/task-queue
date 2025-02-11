// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0
// source: task.sql

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const taskComplete = `-- name: TaskComplete :one
WITH locked_task AS (
  SELECT id
  FROM task
  WHERE id = $1::bigint 
  FOR UPDATE
)
UPDATE task
SET
  state = 'completed'::task_state,
  ended_at = now()
FROM locked_task
WHERE task.id = locked_task.id AND task.state = 'running'
RETURNING task.id, task.state, task.attempt, task.attempted_at, task.created_at, task.ended_at, task.scheduled_at, task.priority, task.payload, task.error, task.kind, task.queue
`

func (q *Queries) TaskComplete(ctx context.Context, id int64) (Task, error) {
	row := q.db.QueryRow(ctx, taskComplete, id)
	var i Task
	err := row.Scan(
		&i.ID,
		&i.State,
		&i.Attempt,
		&i.AttemptedAt,
		&i.CreatedAt,
		&i.EndedAt,
		&i.ScheduledAt,
		&i.Priority,
		&i.Payload,
		&i.Error,
		&i.Kind,
		&i.Queue,
	)
	return i, err
}

const taskFail = `-- name: TaskFail :one
WITH locked_task AS (
  SELECT id
  FROM task
  WHERE id = $2::bigint 
  FOR UPDATE
)
UPDATE task
SET
  state = 'failed'::task_state,
  error = $1::text,
  ended_at = now()
FROM locked_task
WHERE task.id = locked_task.id AND task.state = 'running'
RETURNING task.id, task.state, task.attempt, task.attempted_at, task.created_at, task.ended_at, task.scheduled_at, task.priority, task.payload, task.error, task.kind, task.queue
`

type TaskFailParams struct {
	Err string
	ID  int64
}

func (q *Queries) TaskFail(ctx context.Context, arg TaskFailParams) (Task, error) {
	row := q.db.QueryRow(ctx, taskFail, arg.Err, arg.ID)
	var i Task
	err := row.Scan(
		&i.ID,
		&i.State,
		&i.Attempt,
		&i.AttemptedAt,
		&i.CreatedAt,
		&i.EndedAt,
		&i.ScheduledAt,
		&i.Priority,
		&i.Payload,
		&i.Error,
		&i.Kind,
		&i.Queue,
	)
	return i, err
}

const taskInsert = `-- name: TaskInsert :one
INSERT INTO task(
    state,
    payload,
    kind,
    priority,
    queue
) VALUES (
    'available'::task_state,
    $1::jsonb,
    $2::text,
    $3::int,
    $4::text
) RETURNING id, state, attempt, attempted_at, created_at, ended_at, scheduled_at, priority, payload, error, kind, queue
`

type TaskInsertParams struct {
	Payload  []byte
	Kind     string
	Priority int32
	Queue    string
}

func (q *Queries) TaskInsert(ctx context.Context, arg TaskInsertParams) (Task, error) {
	row := q.db.QueryRow(ctx, taskInsert,
		arg.Payload,
		arg.Kind,
		arg.Priority,
		arg.Queue,
	)
	var i Task
	err := row.Scan(
		&i.ID,
		&i.State,
		&i.Attempt,
		&i.AttemptedAt,
		&i.CreatedAt,
		&i.EndedAt,
		&i.ScheduledAt,
		&i.Priority,
		&i.Payload,
		&i.Error,
		&i.Kind,
		&i.Queue,
	)
	return i, err
}

const taskListAvailable = `-- name: TaskListAvailable :many
WITH
  locked_tasks AS (
    SELECT id, state, attempt, attempted_at, created_at, ended_at, scheduled_at, priority, payload, error, kind, queue
    FROM task
    WHERE
      state = 'available'
      AND queue = $1::text
      AND scheduled_at <= now()
    ORDER BY
      priority ASC,
      scheduled_at ASC,
      id ASC
    LIMIT $2::integer
    FOR UPDATE SKIP LOCKED
  )
UPDATE task
SET
  state = 'running',
  attempt = task.attempt + 1,
  attempted_at = now()
FROM locked_tasks
WHERE task.id = locked_tasks.id
RETURNING task.id, task.state, task.attempt, task.attempted_at, task.created_at, task.ended_at, task.scheduled_at, task.priority, task.payload, task.error, task.kind, task.queue
`

type TaskListAvailableParams struct {
	Queue string
	Max   int32
}

func (q *Queries) TaskListAvailable(ctx context.Context, arg TaskListAvailableParams) ([]Task, error) {
	rows, err := q.db.Query(ctx, taskListAvailable, arg.Queue, arg.Max)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Task
	for rows.Next() {
		var i Task
		if err := rows.Scan(
			&i.ID,
			&i.State,
			&i.Attempt,
			&i.AttemptedAt,
			&i.CreatedAt,
			&i.EndedAt,
			&i.ScheduledAt,
			&i.Priority,
			&i.Payload,
			&i.Error,
			&i.Kind,
			&i.Queue,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const taskRetry = `-- name: TaskRetry :one
WITH locked_task AS (
  SELECT id
  FROM task
  WHERE id = $3::bigint 
  FOR UPDATE
)
UPDATE task
SET
  state = 'available'::task_state,
  error = $1::text,
  scheduled_at = $2::timestamptz
FROM locked_task
WHERE task.id = locked_task.id AND task.state = 'running'
RETURNING task.id, task.state, task.attempt, task.attempted_at, task.created_at, task.ended_at, task.scheduled_at, task.priority, task.payload, task.error, task.kind, task.queue
`

type TaskRetryParams struct {
	Err         string
	Scheduledat pgtype.Timestamptz
	ID          int64
}

func (q *Queries) TaskRetry(ctx context.Context, arg TaskRetryParams) (Task, error) {
	row := q.db.QueryRow(ctx, taskRetry, arg.Err, arg.Scheduledat, arg.ID)
	var i Task
	err := row.Scan(
		&i.ID,
		&i.State,
		&i.Attempt,
		&i.AttemptedAt,
		&i.CreatedAt,
		&i.EndedAt,
		&i.ScheduledAt,
		&i.Priority,
		&i.Payload,
		&i.Error,
		&i.Kind,
		&i.Queue,
	)
	return i, err
}
