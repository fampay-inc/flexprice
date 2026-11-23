package ent

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/flexprice/flexprice/internal/types"
)

// ApplyBaseFilters applies common filters like tenant ID and status using the query options
func ApplyBaseFilters[T any](ctx context.Context, query T, filter types.BaseFilter, opts BaseQueryOptions[T]) T {
	query = opts.ApplyTenantFilter(ctx, query)
	query = opts.ApplyStatusFilter(query, filter.GetStatus())
	query = opts.ApplyEnvironmentFilter(ctx, query)
	return query
}

// ApplyPagination applies pagination to the query if the filter is not unlimited
func ApplyPagination[T any](query T, filter types.BaseFilter, opts BaseQueryOptions[T]) T {
	if filter == nil {
		return query
	}

	if !filter.IsUnlimited() && filter.GetLimit() > 0 {
		query = opts.ApplyPaginationFilter(query, filter.GetLimit(), filter.GetOffset())
	}
	return query
}

// ApplySorting applies sorting to the query
func ApplySorting[T any](query T, filter types.BaseFilter, opts BaseQueryOptions[T]) T {
	if filter == nil {
		// Apply default sorting if filter is nil
		return opts.ApplySortFilter(query, "created_at", "desc")
	}
	return opts.ApplySortFilter(query, filter.GetSort(), filter.GetOrder())
}

// ApplyQueryOptions applies all common query options (base filters, pagination, sorting)
func ApplyQueryOptions[T any](ctx context.Context, query T, filter types.BaseFilter, opts BaseQueryOptions[T]) T {
	query = ApplyBaseFilters(ctx, query, filter, opts)
	if filter == nil {
		return query
	}

	query = ApplySorting(query, filter, opts)
	return ApplyPagination(query, filter, opts)
}

// CursorWhere returns a selector predicate restricting rows to those strictly after the cursor
// position in its order and direction. It is a no-op on the first page.
// Convert it to the entity's predicate type, e.g. predicate.Subscription(CursorWhere(c)).
func CursorWhere(c *types.Cursor) func(*sql.Selector) {
	return func(s *sql.Selector) {
		if c == nil || c.IsFirstPage() {
			return
		}

		switch c.Order {
		case types.CursorOrderID:
			if c.IsDesc() {
				s.Where(sql.LT(s.C("id"), c.LastSeenID))
			} else {
				s.Where(sql.GT(s.C("id"), c.LastSeenID))
			}
		case types.CursorOrderCreatedAt:
			cols := []string{s.C("created_at"), s.C("id")}
			if c.IsDesc() {
				s.Where(sql.CompositeLT(cols, *c.LastSeenCreatedAt, c.LastSeenID))
			} else {
				s.Where(sql.CompositeGT(cols, *c.LastSeenCreatedAt, c.LastSeenID))
			}
		}
	}
}

// CursorOrderBy returns the ORDER BY matching the cursor. It must replace any other sorting
// on the query, otherwise pages skip or repeat rows.
func CursorOrderBy(c *types.Cursor) func(*sql.Selector) {
	return func(s *sql.Selector) {
		if c == nil {
			return
		}

		dir := sql.Asc
		if c.IsDesc() {
			dir = sql.Desc
		}

		switch c.Order {
		case types.CursorOrderID:
			s.OrderBy(dir(s.C("id")))
		case types.CursorOrderCreatedAt:
			s.OrderBy(dir(s.C("created_at")), dir(s.C("id")))
		}
	}
}
