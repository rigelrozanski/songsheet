package main

func getLasses(lines []string) (lasses lineAndSasses) {
	el := sine{} // dummy element to make the call
	for yI := 0; yI < len(lines); yI++ {
		workingLines := lines[yI:]
		_, sasEl, err := el.parseText(workingLines)
		if err == nil {
			sas := sasEl.(sine)
			ls := lineAndSas{int16(yI), sas}
			lasses = append(lasses, ls)
		}
	}
	return lasses
}

var (
	// line offset from the top of a text-sine (with chords and
	// everything) to the middle of the text-based-sine-curve
	lineNoOffsetToMiddleHump = 3

	charsToaHump = 4.0 // 4 character positions to a hump in a text-based sine wave
)

type lineAndSas struct {
	lineNo int16
	sas    sine
}

type lineAndSasses []lineAndSas

// gets the next position of the cursor which is moving hump-movements
// along the text-sine-curve. If there are not enough positions within
// the current hump, then recurively call for the next hump
func (ls lineAndSasses) getNextPosition(startCurX, startHumpIndex, charMovement int) (
	endCurX, endCurY int, endReached bool) {

	if startHumpIndex >= len(ls) {
		return 0, 0, true
	}

	// get the current line hump
	clh := ls[startHumpIndex].sas.totalHumps()

	if startCurX+charMovement <= int(clh*charsToaHump) {
		endCurX = startCurX + charMovement
		endCurY = int(ls[startHumpIndex].lineNo) + lineNoOffsetToMiddleHump
		return endCurX, endCurY, false
	}

	reducedCM := (startCurX + charMovement - int(clh*charsToaHump))
	return ls.getNextPosition(0, startHumpIndex+1, reducedCM)
}
