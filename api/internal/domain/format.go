package domain

import "fmt"

// FormatReach renders lakh as the Indian-unit string used everywhere in the
// UI: >=100L rolls up into Cr, otherwise stays in L.
func FormatReach(lakh float64) string {
	if lakh >= 100 {
		return fmt.Sprintf("%.2fCr", lakh/100)
	}
	return fmt.Sprintf("%.1fL", lakh)
}

// FormatMoney renders whole rupees as the Indian-unit string used
// everywhere in the UI: Cr >= 1,00,00,000; L >= 1,00,000; K >= 1,000;
// otherwise the plain rupee amount.
func FormatMoney(rupees int64) string {
	const (
		crore    = 1_00_00_000
		lakh     = 1_00_000
		thousand = 1_000
	)
	switch {
	case rupees >= crore:
		return fmt.Sprintf("₹%.2fCr", float64(rupees)/crore)
	case rupees >= lakh:
		return fmt.Sprintf("₹%.1fL", float64(rupees)/lakh)
	case rupees >= thousand:
		return fmt.Sprintf("₹%.0fK", float64(rupees)/thousand)
	default:
		return fmt.Sprintf("₹%d", rupees)
	}
}

// FormatIndex renders an index multiple as e.g. "1.4x".
func FormatIndex(index float64) string {
	return fmt.Sprintf("%.1fx", index)
}

// FormatCPM renders a CPM value, or "—" when there's nothing to show.
func FormatCPM(cpm float64) string {
	if cpm <= 0 {
		return "—"
	}
	return fmt.Sprintf("₹%.0f", cpm)
}
