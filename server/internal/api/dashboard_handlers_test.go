package api_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expensetracker/internal/api"
)

func TestDashboardReflectsTransactions(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)
	expenseCat := createTestCategory(t, deps, router, userID, "Groceries", "expense")
	incomeCat := createTestCategory(t, deps, router, userID, "Salary", "income")
	today := time.Now().Format("2006-01-02")

	for _, tc := range []struct {
		categoryID int64
		typ        string
		amount     int
	}{
		{expenseCat, "expense", 30000},
		{incomeCat, "income", 100000},
	} {
		req := authedRequest(t, deps, http.MethodPost, "/api/transactions", userID)
		req.Body = jsonBody(t, map[string]any{
			"categoryId": tc.categoryID, "amount": tc.amount, "type": tc.typ,
			"occurredOn": today, "description": "",
		})
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("seed POST /api/transactions (%s) = %d %s", tc.typ, rec.Code, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authedRequest(t, deps, http.MethodGet, "/api/dashboard", userID))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/dashboard = %d %s, want 200", rec.Code, rec.Body.String())
	}

	dash := decodeJSON[struct {
		TotalExpense      int64 `json:"totalExpense"`
		TotalIncome       int64 `json:"totalIncome"`
		CurrentMonthEmpty bool  `json:"currentMonthEmpty"`
		HeaderBalance     struct {
			Remaining int64 `json:"remaining"`
			Empty     bool  `json:"empty"`
		} `json:"headerBalance"`
		Pie struct {
			Legend []struct {
				Name    string `json:"name"`
				Percent int    `json:"percent"`
				Amount  int64  `json:"amount"`
			} `json:"legend"`
		} `json:"pie"`
		Bar struct {
			Labels  []string `json:"labels"`
			Expense []int64  `json:"expense"`
			Income  []int64  `json:"income"`
		} `json:"bar"`
	}](t, rec)

	if dash.TotalExpense != 30000 {
		t.Errorf("TotalExpense = %d, want 30000", dash.TotalExpense)
	}
	if dash.TotalIncome != 100000 {
		t.Errorf("TotalIncome = %d, want 100000", dash.TotalIncome)
	}
	if dash.CurrentMonthEmpty {
		t.Error("CurrentMonthEmpty = true, want false (transactions were just added)")
	}
	if dash.HeaderBalance.Remaining != 70000 {
		t.Errorf("HeaderBalance.Remaining = %d, want 70000 (100000 income - 30000 expense)", dash.HeaderBalance.Remaining)
	}
	if dash.HeaderBalance.Empty {
		t.Error("HeaderBalance.Empty = true, want false")
	}

	foundGroceries := false
	for _, entry := range dash.Pie.Legend {
		if entry.Name == "Groceries" {
			foundGroceries = true
			if entry.Amount != 30000 {
				t.Errorf("pie legend Groceries amount = %d, want 30000", entry.Amount)
			}
			if entry.Percent != 100 {
				t.Errorf("pie legend Groceries percent = %d, want 100 (only expense category)", entry.Percent)
			}
		}
	}
	if !foundGroceries {
		t.Errorf("pie legend has no Groceries entry: %+v", dash.Pie.Legend)
	}

	if len(dash.Bar.Labels) != 4 {
		t.Errorf("bar series has %d points, want 4 (barMonths)", len(dash.Bar.Labels))
	}
	lastExpense := dash.Bar.Expense[len(dash.Bar.Expense)-1]
	lastIncome := dash.Bar.Income[len(dash.Bar.Income)-1]
	if lastExpense != 30000 || lastIncome != 100000 {
		t.Errorf("bar series' current (last) month = expense %d income %d, want 30000/100000", lastExpense, lastIncome)
	}
}

func TestDashboardEmptyMonthReportsEmpty(t *testing.T) {
	deps := newTestDeps(t)
	router := api.NewRouter(deps)
	userID := createTestUser(t, deps)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, authedRequest(t, deps, http.MethodGet, "/api/dashboard", userID))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/dashboard = %d %s, want 200", rec.Code, rec.Body.String())
	}
	dash := decodeJSON[struct {
		CurrentMonthEmpty bool                 `json:"currentMonthEmpty"`
		HeaderBalance     struct{ Empty bool } `json:"headerBalance"`
	}](t, rec)
	if !dash.CurrentMonthEmpty {
		t.Error("CurrentMonthEmpty = false for a brand-new account, want true")
	}
	if !dash.HeaderBalance.Empty {
		t.Error("HeaderBalance.Empty = false for a brand-new account, want true")
	}

	// Regression test for a real bug a browser smoke test caught: a nil Go
	// slice (the zero value buildPieData/buildBarSeries/monthOptions would
	// produce for an account with no data at all) marshals to JSON null,
	// not []. The client indexes straight into these arrays with no null
	// check (matching every other array field this API returns), so a
	// brand-new account's very first dashboard load crashed outright.
	// Checking the raw body is deliberate: unmarshaling "null" into a Go
	// []T field also reads back as nil, which would hide the regression
	// from a struct-shaped assertion the way dash's decodeJSON above does.
	for _, field := range []string{
		`"availableMonths":null`,
		`"labels":null`, `"values":null`, `"colors":null`, `"legend":null`,
		`"expense":null`, `"income":null`,
	} {
		if strings.Contains(rec.Body.String(), field) {
			t.Errorf("response body contains %q -- an empty array field serialized as JSON null instead of []\nbody: %s", field, rec.Body.String())
		}
	}
}
