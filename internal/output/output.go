package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/rodaine/table"
)

// Format represents the output format.
type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

// ParseFormat validates and returns a Format.
func ParseFormat(s string) (Format, error) {
	switch Format(s) {
	case FormatJSON, FormatTable:
		return Format(s), nil
	default:
		return "", fmt.Errorf("invalid output format %q (use \"json\" or \"table\")", s)
	}
}

// PrintJSON writes data as indented JSON to w.
func PrintJSON(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// PrintTable writes tabular data. headers are column names, rows is a slice of row slices.
func PrintTable(w io.Writer, headers []string, rows [][]string) {
	ifaces := make([]interface{}, len(headers))
	for i, h := range headers {
		ifaces[i] = h
	}
	tbl := table.New(ifaces...).WithWriter(w)
	for _, row := range rows {
		rowIfaces := make([]interface{}, len(row))
		for i, v := range row {
			rowIfaces[i] = v
		}
		tbl.AddRow(rowIfaces...)
	}
	tbl.Print()
}

// Print dispatches to JSON or Table output based on format.
// For table output, the caller must provide headers and a toRow function.
func Print(format Format, data any, headers []string, toRows func(data any) [][]string) error {
	switch format {
	case FormatJSON:
		return PrintJSON(os.Stdout, data)
	case FormatTable:
		rows := toRows(data)
		PrintTable(os.Stdout, headers, rows)
		return nil
	default:
		return PrintJSON(os.Stdout, data)
	}
}
