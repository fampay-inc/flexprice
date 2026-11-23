package types

import (
	"time"

	ierr "github.com/flexprice/flexprice/internal/errors"
	"github.com/samber/lo"
)

// CursorOrder is the fixed column tuple a keyset cursor pages over. The cursor owns the
// ordering: when a cursor is set, the query is sorted by these columns and nothing else.
// Every order ends with id so rows sharing a leading value are never skipped or repeated.
type CursorOrder string

const (
	// CursorOrderID pages over (id). IDs are UUIDv7, so this is unique, indexed and roughly time-ordered.
	CursorOrderID CursorOrder = "id"
	// CursorOrderCreatedAt pages over (created_at, id).
	CursorOrderCreatedAt CursorOrder = "created_at,id"
)

// Cursor marks a position in a keyset-paginated scan. Leave the LastSeen* fields empty
// for the first page; after each page set them from the last row returned.
type Cursor struct {
	Order     CursorOrder `json:"order"`
	Direction string      `json:"direction"`

	// LastSeenID is the id of the last row of the previous page.
	LastSeenID string `json:"last_seen_id,omitempty"`
	// LastSeenCreatedAt is the created_at of the last row of the previous page.
	// Required alongside LastSeenID for CursorOrderCreatedAt.
	LastSeenCreatedAt *time.Time `json:"last_seen_created_at,omitempty"`
}

// NewCursor returns a first-page cursor for the given order and direction.
func NewCursor(order CursorOrder, direction string) *Cursor {
	return &Cursor{Order: order, Direction: direction}
}

// IsFirstPage reports whether the cursor has no position yet.
func (c *Cursor) IsFirstPage() bool {
	return c.LastSeenID == ""
}

// IsDesc reports whether the scan runs in descending order.
func (c *Cursor) IsDesc() bool {
	return c.Direction == OrderDesc
}

// Validate checks the cursor is well-formed and its order is one the caller supports.
// Pass the orders backed by an index on the target table.
func (c *Cursor) Validate(allowed ...CursorOrder) error {
	if !lo.Contains(allowed, c.Order) {
		return ierr.NewError("unsupported cursor order").
			WithHintf("Cursor order must be one of %v", allowed).
			Mark(ierr.ErrValidation)
	}

	if c.Direction != OrderAsc && c.Direction != OrderDesc {
		return ierr.NewError("cursor direction must be either 'asc' or 'desc'").
			Mark(ierr.ErrValidation)
	}

	if c.Order == CursorOrderCreatedAt && (c.LastSeenID == "") != (c.LastSeenCreatedAt == nil) {
		return ierr.NewError("last_seen_id and last_seen_created_at must be set together").
			Mark(ierr.ErrValidation)
	}

	return nil
}

// Advance moves the cursor to the given row, the last one of the page just read.
func (c *Cursor) Advance(id string, createdAt time.Time) {
	c.LastSeenID = id
	if c.Order == CursorOrderCreatedAt {
		c.LastSeenCreatedAt = lo.ToPtr(createdAt)
	}
}
