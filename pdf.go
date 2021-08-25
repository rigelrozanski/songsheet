package main

import "github.com/jung-kurt/gofpdf"

type Pdf interface {
	SetLineWidth(width float64)
	SetLineCapStyle(styleStr string)
	Polygon(points []gofpdf.PointType, styleStr string)
	Line(x1, y1, x2, y2 float64)
	SetFont(familyStr, styleStr string, size float64)
	Text(x, y float64, txtStr string)
	Circle(x, y, r float64, styleStr string)
	Curve(x0, y0, cx, cy, x1, y1 float64, styleStr string)
}

// dummyPdf fulfills the interface Pdf (DNETL)
type dummyPdf struct{}

var _ Pdf = dummyPdf{}

func (d dummyPdf) SetLineWidth(width float64)                            {}
func (d dummyPdf) SetLineCapStyle(styleStr string)                       {}
func (d dummyPdf) Polygon(points []gofpdf.PointType, styleStr string)    {}
func (d dummyPdf) Line(x1, y1, x2, y2 float64)                           {}
func (d dummyPdf) SetFont(familyStr, styleStr string, size float64)      {}
func (d dummyPdf) Text(x, y float64, txtStr string)                      {}
func (d dummyPdf) Circle(x, y, r float64, styleStr string)               {}
func (d dummyPdf) Curve(x0, y0, cx, cy, x1, y1 float64, styleStr string) {}
