package main

const ( // empirically determined
	ptToHeight    = 100  //72
	widthToHeight = 0.82 //
)

func GetFontPt(heightInches float64) float64 {
	return heightInches * ptToHeight
}

func GetFontHeight(fontPt float64) (heightInches float64) {
	return fontPt / ptToHeight
}

func GetCourierFontWidthFromHeight(height float64) float64 {
	return widthToHeight * height
}

func GetCourierFontHeightFromWidth(width float64) float64 {
	return width / widthToHeight
}
