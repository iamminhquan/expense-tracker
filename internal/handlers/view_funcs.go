package handlers

import (
	"html/template"

	"expensetracker/internal/format"
	"expensetracker/internal/i18n"
)

// TemplateFuncs returns the FuncMap every page template needs. The
// formatting rules themselves live in internal/format; this is only the
// mapping from the names the templates call to the functions behind them,
// which is why it stays in the package that owns the templates.
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"vnd":        format.VND,
		"vndSigned":  format.VNDSigned,
		"vndBalance": format.VNDBalance,
		"dateShort":  format.DateShort,
		"catName":    i18n.CategoryName,
		"countOf":    format.CountOf,
		"swatches":   func() []string { return categorySwatches },
	}
}
