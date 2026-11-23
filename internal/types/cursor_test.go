package types

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestCursor_Validate(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name    string
		cursor  Cursor
		allowed []CursorOrder
		wantErr bool
	}{
		{
			name:    "first page by id",
			cursor:  Cursor{Order: CursorOrderID, Direction: OrderAsc},
			allowed: []CursorOrder{CursorOrderID},
		},
		{
			name:    "next page by id desc",
			cursor:  Cursor{Order: CursorOrderID, Direction: OrderDesc, LastSeenID: "a"},
			allowed: []CursorOrder{CursorOrderID},
		},
		{
			name:    "order not allowed",
			cursor:  Cursor{Order: CursorOrderCreatedAt, Direction: OrderAsc},
			allowed: []CursorOrder{CursorOrderID},
			wantErr: true,
		},
		{
			name:    "invalid direction",
			cursor:  Cursor{Order: CursorOrderID, Direction: "up"},
			allowed: []CursorOrder{CursorOrderID},
			wantErr: true,
		},
		{
			name:    "created_at with both positions",
			cursor:  Cursor{Order: CursorOrderCreatedAt, Direction: OrderAsc, LastSeenID: "a", LastSeenCreatedAt: &now},
			allowed: []CursorOrder{CursorOrderCreatedAt},
		},
		{
			name:    "created_at missing timestamp",
			cursor:  Cursor{Order: CursorOrderCreatedAt, Direction: OrderAsc, LastSeenID: "a"},
			allowed: []CursorOrder{CursorOrderCreatedAt},
			wantErr: true,
		},
		{
			name:    "created_at missing id",
			cursor:  Cursor{Order: CursorOrderCreatedAt, Direction: OrderAsc, LastSeenCreatedAt: &now},
			allowed: []CursorOrder{CursorOrderCreatedAt},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cursor.Validate(tt.allowed...)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestCursor_Advance(t *testing.T) {
	createdAt := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	byID := NewCursor(CursorOrderID, OrderAsc)
	assert.True(t, byID.IsFirstPage())
	byID.Advance("id-1", createdAt)
	assert.False(t, byID.IsFirstPage())
	assert.Equal(t, "id-1", byID.LastSeenID)
	assert.Nil(t, byID.LastSeenCreatedAt)

	byCreatedAt := NewCursor(CursorOrderCreatedAt, OrderDesc)
	byCreatedAt.Advance("id-2", createdAt)
	assert.Equal(t, "id-2", byCreatedAt.LastSeenID)
	assert.Equal(t, createdAt, lo.FromPtr(byCreatedAt.LastSeenCreatedAt))
}

func TestSubscriptionFilter_ValidateCursor(t *testing.T) {
	tests := []struct {
		name    string
		filter  SubscriptionFilter
		wantErr bool
	}{
		{
			name: "cursor with limit",
			filter: SubscriptionFilter{
				QueryFilter: &QueryFilter{Limit: lo.ToPtr(100)},
				Cursor:      NewCursor(CursorOrderID, OrderAsc),
			},
		},
		{
			name: "cursor with zero offset",
			filter: SubscriptionFilter{
				QueryFilter: &QueryFilter{Limit: lo.ToPtr(100), Offset: lo.ToPtr(0)},
				Cursor:      NewCursor(CursorOrderID, OrderAsc),
			},
		},
		{
			name: "cursor with offset",
			filter: SubscriptionFilter{
				QueryFilter: &QueryFilter{Offset: lo.ToPtr(10)},
				Cursor:      NewCursor(CursorOrderID, OrderAsc),
			},
			wantErr: true,
		},
		{
			name: "cursor with query sort",
			filter: SubscriptionFilter{
				QueryFilter: &QueryFilter{Sort: lo.ToPtr("created_at")},
				Cursor:      NewCursor(CursorOrderID, OrderAsc),
			},
			wantErr: true,
		},
		{
			name: "cursor with query order",
			filter: SubscriptionFilter{
				QueryFilter: &QueryFilter{Order: lo.ToPtr(OrderDesc)},
				Cursor:      NewCursor(CursorOrderID, OrderAsc),
			},
			wantErr: true,
		},
		{
			name: "cursor with dsl sort",
			filter: SubscriptionFilter{
				Sort:   []*SortCondition{{Field: "created_at"}},
				Cursor: NewCursor(CursorOrderID, OrderAsc),
			},
			wantErr: true,
		},
		{
			name: "created_at order not index-backed for subscriptions",
			filter: SubscriptionFilter{
				Cursor: NewCursor(CursorOrderCreatedAt, OrderAsc),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
