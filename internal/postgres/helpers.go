package postgres

import (
	"strconv"
	"strings"
)

// placeholder returns "$n" for building dynamic queries.
func placeholder(n int) string { return "$" + strconv.Itoa(n) }

// prefixCols prefixes each column in a comma-separated list with an alias,
// e.g. prefixCols("id, name", "t") => "t.id, t.name".
func prefixCols(cols, alias string) string {
	parts := strings.Split(cols, ",")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}

// intsFromInt16 converts a pgx-scanned []int16 (SMALLINT[]) into []int.
func intsFromInt16(in []int16) []int {
	out := make([]int, len(in))
	for i, v := range in {
		out[i] = int(v)
	}
	return out
}

// int16sFromInts converts []int into []int16 for SMALLINT[] parameters.
func int16sFromInts(in []int) []int16 {
	out := make([]int16, len(in))
	for i, v := range in {
		out[i] = int16(v)
	}
	return out
}
