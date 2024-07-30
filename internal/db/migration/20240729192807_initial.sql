-- migrate:up
CREATE TYPE task_state AS ENUM('available', 'failed', 'completed', 'running');

CREATE TABLE task (
  id bigint generated always as identity PRIMARY KEY,
  state task_state NOT NULL DEFAULT 'available',
  attempt smallint NOT NULL DEFAULT 0,

  attempted_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  ended_at timestamptz,
  scheduled_at timestamptz NOT NULL DEFAULT NOW(),

  priority smallint NOT NULL DEFAULT 0,
  payload jsonb,
  error text,
  kind text NOT NULL,
  queue text NOT NULL
);

CREATE INDEX idx_task_fetch ON task (state, queue, scheduled_at, priority, id);

CREATE INDEX idx_task_id_state ON task (id, state);


-- migrate:down

