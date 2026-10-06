package api

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"expensetracker/internal/i18n"
	"expensetracker/internal/pgval"
	"expensetracker/internal/sqlcgen"
	"expensetracker/internal/txnrule"

	"github.com/gin-gonic/gin"
)

// transactionDTO is one transaction as the client sees it. OccurredOn is
// "2006-01-02" (an <input type="date"> value, not a display string --
// handlers.rowDate's long/short-form display formatting is a presentation
// decision the client makes, not something this API bakes in). IsDuplicate
// mirrors handlers/txn_row.go's markDuplicates: it only ever compares rows
// within the same page of results, the same accepted trade documented
// there.
type transactionDTO struct {
	ID            int64  `json:"id"`
	CategoryID    int64  `json:"categoryId"`
	CategoryName  string `json:"categoryName"`
	CategoryColor string `json:"categoryColor"`
	Description   string `json:"description"`
	Amount        int64  `json:"amount"`
	Type          string `json:"type"`
	OccurredOn    string `json:"occurredOn"`
	IsDuplicate   bool   `json:"isDuplicate"`
}

func newTransactionDTO(row sqlcgen.ListTransactionsForMonthRow) transactionDTO {
	return transactionDTO{
		ID:            row.ID,
		CategoryID:    row.CategoryID,
		CategoryName:  i18n.CategoryName(row.CategorySlug, row.CategoryName),
		CategoryColor: row.CategoryColor,
		Description:   row.Description,
		Amount:        row.Amount,
		Type:          row.Type,
		OccurredOn:    row.OccurredOn.Time.Format("2006-01-02"),
	}
}

// markDuplicates mirrors handlers/txn_row.go's function of the same name:
// flags every row sharing its date, amount, and type with another row in
// the same slice.
func markDuplicates(rows []transactionDTO) {
	type key struct {
		date   string
		amount int64
		typ    string
	}
	byKey := make(map[key][]int, len(rows))
	for i, r := range rows {
		byKey[key{date: r.OccurredOn, amount: r.Amount, typ: r.Type}] = append(byKey[key{date: r.OccurredOn, amount: r.Amount, typ: r.Type}], i)
	}
	for _, idxs := range byKey {
		if len(idxs) < 2 {
			continue
		}
		for _, i := range idxs {
			rows[i].IsDuplicate = true
		}
	}
}

// listTransactionsResponse is GET /api/v1/transactions' body: the requested
// page of rows, enough to draw pagination, and the month-picker state the
// transactions page needs (which months have data, which one is "now").
type listTransactionsResponse struct {
	Transactions      []transactionDTO `json:"transactions"`
	TotalCount        int64            `json:"totalCount"`
	Page              int              `json:"page"`
	TotalPages        int              `json:"totalPages"`
	HasPrev           bool             `json:"hasPrev"`
	HasNext           bool             `json:"hasNext"`
	MonthValue        string           `json:"monthValue"`
	MonthLabel        string           `json:"monthLabel"`
	AllMonths         bool             `json:"allMonths"`
	CurrentMonthValue string           `json:"currentMonthValue"`
	AvailableMonths   []monthOptionDTO `json:"availableMonths"`
}

func listTransactionsHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		query := c.Request.URL.Query()
		filters := filtersFromQuery(query)
		scope := newMonthScope(query.Get("month"))
		from, to := scope.Bounds()

		count, err := deps.Queries.CountTransactionsForMonth(c.Request.Context(), filters.countParams(userID, from, to))
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not load transactions")
			return
		}
		pgr := newPager(pageParam(query.Get("page")), count)

		rows, err := deps.Queries.ListTransactionsForMonth(c.Request.Context(), filters.listParams(userID, from, to, pgr.offset()))
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not load transactions")
			return
		}
		dtos := make([]transactionDTO, len(rows))
		for i, row := range rows {
			dtos[i] = newTransactionDTO(row)
		}
		markDuplicates(dtos)

		months, err := deps.Queries.ListDistinctTransactionMonths(c.Request.Context(), userID)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "could not load transactions")
			return
		}
		currentFrom, _ := currentMonthRange()

		respondSuccess(c, http.StatusOK, "transactions retrieved", listTransactionsResponse{
			Transactions:      dtos,
			TotalCount:        count,
			Page:              pgr.Page,
			TotalPages:        pgr.TotalPages,
			HasPrev:           pgr.HasPrev,
			HasNext:           pgr.HasNext,
			MonthValue:        scope.Value,
			MonthLabel:        scope.Label,
			AllMonths:         scope.All,
			CurrentMonthValue: currentFrom.Time.Format("2006-01"),
			AvailableMonths:   monthOptions(months, currentFrom),
		})
	}
}

// txnWriteRequest is the body shape POST and PATCH /api/v1/transactions both
// bind -- see createTransactionHandler/updateTransactionHandler for which
// fields each actually uses. Type is a *string (not plain string) only so
// create can tell "absent" apart from "": update ignores it entirely,
// since an edit never changes a transaction's type (see
// updateTransactionHandler's comment).
type txnWriteRequest struct {
	CategoryID  int64   `json:"categoryId"`
	Amount      int64   `json:"amount"`
	OccurredOn  string  `json:"occurredOn"`
	Description string  `json:"description"`
	Type        *string `json:"type"`
}

// parsedTxnForm is txnWriteRequest after the strict parse
// handlers.txnFormFromRequest does: a value that fails to parse here never
// came from a client that read this API's contract, so it is a 400 rather
// than a field-level validation message -- mirroring handlers.go's own
// split between parse failures and violation().
type parsedTxnForm struct {
	CategoryID  int64
	Amount      int64
	OccurredOn  time.Time
	Description string
}

func parseTxnForm(req txnWriteRequest) (parsedTxnForm, string) {
	if req.CategoryID <= 0 {
		return parsedTxnForm{}, "invalid category"
	}
	if req.Amount <= 0 {
		return parsedTxnForm{}, "invalid amount"
	}
	occurredOn, err := time.Parse("2006-01-02", req.OccurredOn)
	if err != nil {
		return parsedTxnForm{}, "invalid date"
	}
	return parsedTxnForm{
		CategoryID: req.CategoryID, Amount: req.Amount,
		OccurredOn: occurredOn, Description: req.Description,
	}, ""
}

// violation mirrors handlers.txnForm.violation exactly -- see its comment
// for why these three are checked separately from the parse failures
// above.
func (f parsedTxnForm) violation(categoryType, txnType string) string {
	switch {
	case categoryType != txnType:
		return "that category does not match the transaction type"
	case txnrule.NoteTooLong(f.Description):
		return "note must be " + strconv.Itoa(txnrule.MaxNoteRunes) + " characters or fewer"
	case txnrule.TooFarInFuture(f.OccurredOn, time.Now().In(vietnamLocation)):
		return "that date is too far in the future"
	}
	return ""
}

func createTransactionHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		var req txnWriteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "malformed request body")
			return
		}
		if req.Type == nil || (*req.Type != "expense" && *req.Type != "income") {
			respondError(c, http.StatusBadRequest, "type must be \"expense\" or \"income\"")
			return
		}
		txnType := *req.Type

		form, msg := parseTxnForm(req)
		if msg != "" {
			respondError(c, http.StatusBadRequest, msg)
			return
		}

		category, err := deps.Queries.GetCategoryForUser(c.Request.Context(), sqlcgen.GetCategoryForUserParams{
			ID: form.CategoryID, UserID: pgval.Int64(userID),
		})
		if err != nil {
			respondError(c, http.StatusForbidden, "category not found")
			return
		}
		if v := form.violation(category.Type, txnType); v != "" {
			respondError(c, http.StatusBadRequest, v)
			return
		}

		created, err := deps.Queries.CreateTransaction(c.Request.Context(), sqlcgen.CreateTransactionParams{
			UserID: userID, CategoryID: form.CategoryID, Amount: form.Amount, Type: txnType,
			Description: form.Description, OccurredOn: pgval.Date(form.OccurredOn),
		})
		if err != nil {
			log.Printf("create transaction: %v", err)
			respondError(c, http.StatusInternalServerError, "could not add the transaction, please try again")
			return
		}

		respondSuccess(c, http.StatusCreated, "transaction created", transactionDTO{
			ID: created.ID, CategoryID: created.CategoryID,
			CategoryName: i18n.CategoryName(category.Slug, category.Name), CategoryColor: category.Color,
			Description: created.Description, Amount: created.Amount, Type: created.Type,
			OccurredOn: created.OccurredOn.Time.Format("2006-01-02"),
		})
	}
}

func transactionIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

// updateTransactionHandler never changes a transaction's type -- a row
// keeps the one it was created with (mirrors
// handlers.updateTransactionHandler's identical comment) -- so a type in
// the request body, if sent, is simply ignored rather than rejected.
func updateTransactionHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		id, ok := transactionIDParam(c)
		if !ok {
			return
		}

		existing, err := deps.Queries.GetTransaction(c.Request.Context(), sqlcgen.GetTransactionParams{ID: id, UserID: userID})
		if err != nil {
			respondError(c, http.StatusNotFound, "transaction not found")
			return
		}

		var req txnWriteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "malformed request body")
			return
		}
		form, msg := parseTxnForm(req)
		if msg != "" {
			respondError(c, http.StatusBadRequest, msg)
			return
		}

		category, err := deps.Queries.GetCategoryForUser(c.Request.Context(), sqlcgen.GetCategoryForUserParams{ID: form.CategoryID, UserID: pgval.Int64(userID)})
		if err != nil {
			respondError(c, http.StatusForbidden, "category not found")
			return
		}
		if v := form.violation(category.Type, existing.Type); v != "" {
			respondError(c, http.StatusBadRequest, v)
			return
		}

		updated, err := deps.Queries.UpdateTransaction(c.Request.Context(), sqlcgen.UpdateTransactionParams{
			ID: id, UserID: userID, CategoryID: form.CategoryID, Amount: form.Amount, Type: existing.Type,
			Description: form.Description, OccurredOn: pgval.Date(form.OccurredOn),
		})
		if err != nil {
			log.Printf("update transaction: %v", err)
			respondError(c, http.StatusInternalServerError, "could not update transaction")
			return
		}

		respondSuccess(c, http.StatusOK, "transaction updated", transactionDTO{
			ID: updated.ID, CategoryID: updated.CategoryID,
			CategoryName: i18n.CategoryName(category.Slug, category.Name), CategoryColor: category.Color,
			Description: updated.Description, Amount: updated.Amount, Type: updated.Type,
			OccurredOn: updated.OccurredOn.Time.Format("2006-01-02"),
		})
	}
}

func deleteTransactionHandler(deps Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := UserID(c)
		id, ok := transactionIDParam(c)
		if !ok {
			return
		}
		if _, err := deps.Queries.DeleteTransaction(c.Request.Context(), sqlcgen.DeleteTransactionParams{ID: id, UserID: userID}); err != nil {
			respondError(c, http.StatusInternalServerError, "could not delete transaction")
			return
		}
		respondSuccess[any](c, http.StatusOK, "transaction deleted", nil)
	}
}
