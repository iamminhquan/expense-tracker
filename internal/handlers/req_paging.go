package handlers

import (
	"net/http"
	"strconv"
)

// pageSize is how many transactions one page of the list holds. Change it
// here and the query window, the pager and the page count all follow.
const pageSize = 10

// pager is everything the transactions_pager fragment needs to draw itself.
// It is built by newPager rather than assembled in the handler, so the two
// places that render a page of transactions (the list page and a create that
// bounces back to page 1) cannot disagree about what "page 2 of 3" means.
//
// MonthValue rides along because the pager's own links have to stay inside
// the month being browsed; without it, paging an older month would silently
// jump back to today's.
type pager struct {
	Page       int
	TotalPages int
	PrevPage   int
	NextPage   int
	HasPrev    bool
	HasNext    bool
	MonthValue string
}

// newPager resolves the requested page against how many rows the month
// actually holds. requested is whatever the URL asked for, however nonsensical
// -- 0, negative, past the end -- and is clamped to a page that exists, in the
// same spirit as monthRangeFor's fallback for a malformed month. A month with
// no transactions still has one (empty) page, so TotalPages is never 0 and the
// offset newPager reports is never negative.
func newPager(requested int, total int64, monthValue string) pager {
	totalPages := int((total + pageSize - 1) / pageSize)
	if totalPages < 1 {
		totalPages = 1
	}
	page := requested
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}
	return pager{
		Page:       page,
		TotalPages: totalPages,
		PrevPage:   page - 1,
		NextPage:   page + 1,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		MonthValue: monthValue,
	}
}

// Offset is the row offset the page's query window starts at.
func (p pager) Offset() int32 { return int32((p.Page - 1) * pageSize) }

// pageParam reads a "page" query value. Anything unparseable becomes 0, which
// newPager then clamps up to the first page -- no error path, because a bad
// page number in a URL is not worth an error page.
func pageParam(raw string) int {
	page, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return page
}

// pageFromRequest reports which page the page that issued this request was
// showing, read off HX-Current-URL rather than r.URL -- see currentURLQuery.
// No header means 0 rather than 1, which newPager clamps to the first page
// anyway, and which the "past page 1?" check below reads the same way.
func pageFromRequest(r *http.Request) int {
	return pageParam(currentURLQuery(r).Get("page"))
}
