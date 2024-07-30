-- name: TaskListAvailable :many
WITH
  locked_tasks AS (
    SELECT *
    FROM task
    WHERE
      state = 'available'
      AND queue = sqlc.arg(queue)::text
      AND scheduled_at <= now()
    ORDER BY
      priority ASC,
      scheduled_at ASC,
      id ASC
    LIMIT sqlc.arg(max)::integer
    FOR UPDATE SKIP LOCKED
  )
UPDATE task
SET
  state = 'running',
  attempt = task.attempt + 1,
  attempted_at = now()
FROM locked_tasks
WHERE task.id = locked_tasks.id
RETURNING task.*;


-- name: TaskComplete :one
WITH locked_task AS (
  SELECT id
  FROM task
  WHERE id = sqlc.arg(id)::bigint 
  FOR UPDATE
)
UPDATE task
SET
  state = 'completed'::task_state,
  ended_at = now()
FROM locked_task
WHERE task.id = locked_task.id AND task.state = 'running'
RETURNING task.*;

-- name: TaskFail :one
WITH locked_task AS (
  SELECT id
  FROM task
  WHERE id = sqlc.arg(id)::bigint 
  FOR UPDATE
)
UPDATE task
SET
  state = 'failed'::task_state,
  error = sqlc.arg(err)::text,
  ended_at = now()
FROM locked_task
WHERE task.id = locked_task.id AND task.state = 'running'
RETURNING task.*;

-- name: TaskRetry :one
WITH locked_task AS (
  SELECT id
  FROM task
  WHERE id = sqlc.arg(id)::bigint 
  FOR UPDATE
)
UPDATE task
SET
  state = 'available'::task_state,
  error = sqlc.arg(err)::text,
  scheduled_at = sqlc.arg(scheduledAt)::timestamptz
FROM locked_task
WHERE task.id = locked_task.id AND task.state = 'running'
RETURNING task.*;

-- name: TaskInsert :one
INSERT INTO task(
    state,
    payload,
    kind,
    priority,
    queue
) VALUES (
    'available'::task_state,
    sqlc.arg(payload)::jsonb,
    sqlc.arg(kind)::text,
    sqlc.arg(priority)::int,
    sqlc.arg(queue)::text
) RETURNING *;