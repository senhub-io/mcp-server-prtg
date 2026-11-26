package database

import "html"

// PRTG stores names HTML-escaped in its tables.
func cleanText(s string) string { return html.UnescapeString(s) }
