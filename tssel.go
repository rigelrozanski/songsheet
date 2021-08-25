package main

// whole text songsheet element
type tssElement interface {
	printPDF(Pdf, bounds) (reduced bounds)
	parseText(lines []string) (reducedLines []string, elem tssElement, err error)
}
