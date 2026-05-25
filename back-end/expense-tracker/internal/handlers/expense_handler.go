package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iamminhquan/expense-tracker/internal/models"
	"github.com/iamminhquan/expense-tracker/internal/services"
)

type ExpenseHandler struct {
	service *services.ExpenseService
}

func NewExpenseHandler(service *services.ExpenseService) *ExpenseHandler {
	return &ExpenseHandler{service: service}
}

type expenseRequest struct {
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	Date            string  `json:"date"`
	PaymentMethod   string  `json:"payment_method"`
	IsRecurring     bool    `json:"is_recurring"`
	RecurringPeriod string  `json:"recurring_period"`
	Note            string  `json:"note"`
}

type expenseResponse struct {
	ID              uint    `json:"id"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	Date            string  `json:"date"`
	PaymentMethod   string  `json:"payment_method"`
	IsRecurring     bool    `json:"is_recurring"`
	RecurringPeriod string  `json:"recurring_period"`
	Note            string  `json:"note"`
}

func (h *ExpenseHandler) List(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	expenses, err := h.service.List(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load expenses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"expenses": expenseResponses(expenses)})
}

func (h *ExpenseHandler) Create(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	input, ok := expenseInputFromRequest(c)
	if !ok {
		return
	}

	expense, err := h.service.Create(c.Request.Context(), userID, input)
	if errors.Is(err, services.ErrInvalidExpenseInput) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount, description, category, and date are required"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create expense"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"expense": expenseResponseFromModel(*expense)})
}

func (h *ExpenseHandler) Update(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, ok := expenseIDFromParam(c)
	if !ok {
		return
	}

	input, ok := expenseInputFromRequest(c)
	if !ok {
		return
	}

	expense, err := h.service.Update(c.Request.Context(), userID, id, input)
	if errors.Is(err, services.ErrInvalidExpenseInput) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount, description, category, and date are required"})
		return
	}
	if errors.Is(err, services.ErrExpenseNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "expense not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update expense"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"expense": expenseResponseFromModel(*expense)})
}

func (h *ExpenseHandler) Delete(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id, ok := expenseIDFromParam(c)
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), userID, id); errors.Is(err, services.ErrExpenseNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "expense not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete expense"})
		return
	}

	c.Status(http.StatusNoContent)
}

func expenseInputFromRequest(c *gin.Context) (services.ExpenseInput, bool) {
	var req expenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return services.ExpenseInput{}, false
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date must use YYYY-MM-DD format"})
		return services.ExpenseInput{}, false
	}

	return services.ExpenseInput{
		Amount:          req.Amount,
		Currency:        req.Currency,
		Description:     req.Description,
		Category:        req.Category,
		Date:            date,
		PaymentMethod:   req.PaymentMethod,
		IsRecurring:     req.IsRecurring,
		RecurringPeriod: req.RecurringPeriod,
		Note:            req.Note,
	}, true
}

func expenseIDFromParam(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 0)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense id"})
		return 0, false
	}

	return uint(id), true
}

func userIDFromContext(c *gin.Context) (uint, bool) {
	userIDValue, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}

	userID, ok := userIDValue.(uint)
	return userID, ok
}

func expenseResponses(expenses []models.Expense) []expenseResponse {
	responses := make([]expenseResponse, 0, len(expenses))
	for _, expense := range expenses {
		responses = append(responses, expenseResponseFromModel(expense))
	}

	return responses
}

func expenseResponseFromModel(expense models.Expense) expenseResponse {
	return expenseResponse{
		ID:              expense.ID,
		Amount:          expense.Amount,
		Currency:        expense.Currency,
		Description:     expense.Description,
		Category:        expense.Category,
		Date:            expense.Date.Format("2006-01-02"),
		PaymentMethod:   expense.PaymentMethod,
		IsRecurring:     expense.IsRecurring,
		RecurringPeriod: expense.RecurringPeriod,
		Note:            expense.Note,
	}
}
