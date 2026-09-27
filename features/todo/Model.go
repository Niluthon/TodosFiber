package todo

import (
	"database/sql/driver"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type TodoStatus string

const (
	Pending TodoStatus = "pending"
	Done    TodoStatus = "done"
)

// Value tells GORM how to store TodoStatus in the DB -> as a string.
// This makes it implement driver.Valuer.
func (s TodoStatus) Value() (driver.Value, error) {
	return string(s), nil
}

// Scan tells GORM how to read a DB value into TodoStatus.
// This makes it implement sql.Scanner.
func (s *TodoStatus) Scan(value any) error {
	switch v := value.(type) {
	case nil:
		*s = ""
	case string:
		*s = TodoStatus(v)
	case []byte:
		*s = TodoStatus(v)
	default:
		return fmt.Errorf("todo: unsupported type for TodoStatus: %T", value)
	}
	return nil
}

type TodoGorm struct {
	ID        uint       `json:"id" gorm:"primaryKey"`
	Title     string     `json:"title"`
	Status    TodoStatus `json:"status" gorm:"type:varchar(20);not null;default:pending"`
	DueDate   *time.Time `json:"due_date"`
	CreatedAt time.Time  `json:"created_at"`
}

// BeforeCreate defaults a new record's status to Pending when unset.
func (t *TodoGorm) BeforeCreate(tx *gorm.DB) error {
	if t.Status == "" {
		t.Status = Pending
	}
	return nil
}
