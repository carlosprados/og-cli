package opengate

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
)

// Page size bounds for search requests.
//
// MaxPageSize is what the platform's limit schema documents as the ceiling for
// limit.size; a larger value is rejected by the server. DefaultPageSize is what
// the iterators request when the caller expresses no preference — deliberately
// below the maximum, since a huge page is a large response to buffer and a long
// request to lose on a timeout.
const (
	DefaultPageSize = 1000
	MaxPageSize     = 2000
)

// Page is the pagination block OpenGate returns alongside search results.
//
// Number is the 1-based index of the page being returned. Of is the total number
// of pages, but **not every endpoint reports it** — the alarms search documents
// only number — so treat Of == 0 as "unknown" rather than "no pages".
type Page struct {
	Number int `json:"number,omitempty"`
	Of     int `json:"of,omitempty"`
}

// searchLimit mirrors the limit block of a search body.
//
// Start is a **page number, not an element offset**: the platform documents it
// as "Page number you request. The count starts with number 1". Getting this
// wrong silently re-reads the first N pages instead of walking the result set.
type searchLimit struct {
	Start int `json:"start,omitempty"`
	Size  int `json:"size,omitempty"`
}

// limitOf reports the limit block already present in a search filter. A zero
// value means the filter does not constrain paging.
func limitOf(filter json.RawMessage) searchLimit {
	if len(filter) == 0 {
		return searchLimit{}
	}
	var body struct {
		Limit searchLimit `json:"limit"`
	}
	if json.Unmarshal(filter, &body) != nil {
		return searchLimit{}
	}
	return body.Limit
}

// SetFilterPage returns filter with its limit block set to the requested page,
// preserving every other key. Use it to page a filter you built elsewhere — for
// example JobIDFilter — without hand-editing JSON.
//
// page is a 1-based page number for the filter-based search endpoints. The name
// deliberately avoids the With prefix, which in this package marks the
// functional options accepted by New.
func SetFilterPage(filter json.RawMessage, page, size int) (json.RawMessage, error) {
	return withPage(filter, page, size)
}

// withPage returns filter with limit.start and limit.size set to the requested
// page, preserving every other key. A nil or empty filter yields a body that
// only carries the limit.
func withPage(filter json.RawMessage, page, size int) (json.RawMessage, error) {
	body := make(map[string]any)
	if len(filter) > 0 {
		if err := json.Unmarshal(filter, &body); err != nil {
			return nil, fmt.Errorf("filter is not a JSON object: %w", err)
		}
	}
	body["limit"] = searchLimit{Start: page, Size: size}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("building paged filter: %w", err)
	}
	return data, nil
}

// pageFetcher retrieves one page of results, returning its items and the
// pagination block the server reported.
type pageFetcher[T any] func(ctx context.Context, filter json.RawMessage) ([]T, *Page, error)

// walkPages is the shared paging loop behind every iterator. fetch is called
// with successive page numbers and returns that page's items plus the
// pagination block the server reported. first is the page number to start from.
//
// Iteration stops when the server reports the last page (number >= of), when a
// page comes back empty, or when a page is shorter than requested — the last
// two cover the endpoints that do not report "of" at all.
//
// The next page number follows the number the server reports, falling back to a
// local increment. That matters because the platform is not consistent about
// where paging starts: the search endpoints document limit.start as counting
// from 1, while the job operations endpoint documents its start query parameter
// with a default of 0. Following the server's own numbering works under either
// convention instead of silently skipping or repeating a page.
//
// A cancelled ctx ends the iteration, yielding the context error once so the
// consumer can tell a truncated walk from a complete one.
func walkPages[T any](ctx context.Context, first, size int, fetch func(ctx context.Context, page int) ([]T, *Page, error)) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		var zero T

		for page := first; ; {
			if err := ctx.Err(); err != nil {
				yield(zero, err)
				return
			}

			items, info, err := fetch(ctx, page)
			if err != nil {
				yield(zero, err)
				return
			}

			for _, item := range items {
				if !yield(item, nil) {
					return
				}
			}

			if len(items) == 0 || len(items) < size {
				return
			}
			if info != nil && info.Of > 0 && info.Number > 0 && info.Number >= info.Of {
				return
			}

			if info != nil && info.Number > 0 {
				page = info.Number + 1
			} else {
				page++
			}
		}
	}
}

// paginate walks every page of a filter-based search — a POST whose body carries
// the limit block — and yields items one by one.
//
// The page size comes from the caller's filter when it sets one, so an explicit
// limit.size is honoured rather than overridden; otherwise DefaultPageSize is
// used.
func paginate[T any](ctx context.Context, filter json.RawMessage, fetch pageFetcher[T]) iter.Seq2[T, error] {
	size := limitOf(filter).Size
	if size <= 0 {
		size = DefaultPageSize
	}

	return walkPages(ctx, 1, size, func(ctx context.Context, page int) ([]T, *Page, error) {
		paged, err := withPage(filter, page, size)
		if err != nil {
			return nil, nil, err
		}
		return fetch(ctx, paged)
	})
}
