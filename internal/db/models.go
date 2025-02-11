// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package db

import (
	"database/sql/driver"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

type TaskState string

const (
	TaskStateAvailable TaskState = "available"
	TaskStateFailed    TaskState = "failed"
	TaskStateCompleted TaskState = "completed"
	TaskStateRunning   TaskState = "running"
)

func (e *TaskState) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = TaskState(s)
	case string:
		*e = TaskState(s)
	default:
		return fmt.Errorf("unsupported scan type for TaskState: %T", src)
	}
	return nil
}

type NullTaskState struct {
	TaskState TaskState
	Valid     bool // Valid is true if TaskState is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullTaskState) Scan(value interface{}) error {
	if value == nil {
		ns.TaskState, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.TaskState.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullTaskState) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.TaskState), nil
}

type Task struct {
	ID          int64
	State       TaskState
	Attempt     int16
	AttemptedAt pgtype.Timestamptz
	CreatedAt   pgtype.Timestamptz
	EndedAt     pgtype.Timestamptz
	ScheduledAt pgtype.Timestamptz
	Priority    int16
	Payload     []byte
	Error       pgtype.Text
	Kind        string
	Queue       string
}
