package buffer

import (
	"fmt"
)

type Buffer struct {
	lines                 []Line
	cursorX               uint16
	cursorY               uint16
	viewHeight            uint16
	viewWidth             uint16
	cursorAttr            CellAttributes
	displayChangeHandlers []chan bool
	savedX                uint16
	savedY                uint16
}

// NewBuffer creates a new terminal buffer
func NewBuffer(viewCols uint16, viewLines uint16, attr CellAttributes) *Buffer {
	b := &Buffer{
		cursorX:    0,
		cursorY:    0,
		lines:      []Line{},
		cursorAttr: attr,
	}
	b.ResizeView(viewCols, viewLines)
	return b
}

func (buffer *Buffer) SaveCursor() {
	buffer.savedX = buffer.cursorX
	buffer.savedY = buffer.cursorY
}

func (buffer *Buffer) RestoreCursor() {
	buffer.cursorX = buffer.savedX
	buffer.cursorY = buffer.savedY
}

func (buffer *Buffer) CursorAttr() *CellAttributes {
	return &buffer.cursorAttr
}

func (buffer *Buffer) GetCell(viewCol int, viewRow int) *Cell {

	rawLine := buffer.convertViewLineToRawLine(uint16(viewRow))

	if viewCol < 0 || rawLine < 0 || int(rawLine) >= len(buffer.lines) {
		return nil
	}
	line := &buffer.lines[rawLine]
	if viewCol >= len(line.cells) {
		return nil
	}
	return &line.cells[viewCol]
}

func (buffer *Buffer) AttachDisplayChangeHandler(handler chan bool) {
	if buffer.displayChangeHandlers == nil {
		buffer.displayChangeHandlers = []chan bool{}
	}

	buffer.displayChangeHandlers = append(buffer.displayChangeHandlers, handler)
}

func (buffer *Buffer) emitDisplayChange() {
	for _, channel := range buffer.displayChangeHandlers {
		go func(c chan bool) {
			select {
			case c <- true:
			default:
			}
		}(channel)
	}
}

// Column returns cursor column
func (buffer *Buffer) CursorColumn() uint16 {
	return buffer.cursorX
}

// Line returns cursor line
func (buffer *Buffer) CursorLine() uint16 {
	return buffer.cursorY
}

// translates the cursor line to the raw buffer line
func (buffer *Buffer) RawLine() uint64 {
	return buffer.convertViewLineToRawLine(buffer.cursorY)
}

func (buffer *Buffer) convertViewLineToRawLine(viewLine uint16) uint64 {
	rawHeight := buffer.Height()
	if int(buffer.viewHeight) > rawHeight {
		return uint64(viewLine)
	}
	return uint64(int(viewLine) + (rawHeight - int(buffer.viewHeight)))
}

// Width returns the width of the buffer in columns
func (buffer *Buffer) Width() uint16 {
	return buffer.viewWidth
}

func (buffer *Buffer) ViewWidth() uint16 {
	return buffer.viewWidth
}

func (buffer *Buffer) Height() int {
	return len(buffer.lines)
}

func (buffer *Buffer) ViewHeight() uint16 {
	return buffer.viewHeight
}

// Write will write a rune to the terminal at the position of the cursor, and increment the cursor position
func (buffer *Buffer) Write(runes ...rune) {
	for _, r := range runes {
		if r == 0x0a {
			buffer.NewLine()
			continue
		} else if r == 0x0d {
			buffer.CarriageReturn()
			continue
		}
		line := buffer.getCurrentLine()

		if buffer.CursorColumn() >= buffer.Width() { // if we're after the line, move to next
			buffer.cursorX = 0

			if buffer.cursorY >= buffer.ViewHeight()-1 {
				buffer.lines = append(buffer.lines, newLine())
			} else {
				buffer.cursorY++
			}

			//if int(buffer.RawLine()) >= len(buffer.lines) { // if we need to create a new line, set it to wrapped and write to it

			newLine := buffer.getCurrentLine()
			newLine.setWrapped(true)
			if len(newLine.cells) == 0 {
				newLine.cells = []Cell{Cell{}}
			}
			cell := &newLine.cells[buffer.CursorColumn()]
			cell.setRune(r)
			cell.attr = buffer.cursorAttr
			//} else {
			//newLine := &buffer.lines[buffer.RawLine()]
			//}

			// @todo if next line is wrapped then prepend to it and shuffle characters along line, wrapping to next if necessary
		} else {

			for int(buffer.CursorColumn()) >= len(line.cells) {
				line.cells = append(line.cells, newCell())
			}

			cell := &line.cells[buffer.CursorColumn()]
			cell.setRune(r)
			cell.attr = buffer.cursorAttr

		}

		buffer.incrementCursorPosition()
	}
}

func (buffer *Buffer) incrementCursorPosition() {

	defer buffer.emitDisplayChange()

	// we can increment one column past the end of the line.
	// this is effectively the beginning of the next line, except when we \r etc.
	if buffer.CursorColumn() < buffer.Width() { // if not at end of line

		buffer.cursorX++

	} else {
		panic("bad")
	}
}

func (buffer *Buffer) Backspace() {

	if buffer.cursorX == 0 {
		line := buffer.getCurrentLine()
		if line.wrapped {
			buffer.MovePosition(int16(buffer.Width()-1), -1)
		} else {
			//@todo ring bell or whatever
			fmt.Println("BELL?")
		}
	} else {
		buffer.MovePosition(-1, 0)
	}
}

func (buffer *Buffer) CarriageReturn() {
	defer buffer.emitDisplayChange()
	buffer.cursorX = 0
}

func (buffer *Buffer) NewLine() {
	defer buffer.emitDisplayChange()
	if buffer.cursorY >= buffer.ViewHeight()-1 {
		buffer.lines = append(buffer.lines, newLine())
	} else {
		buffer.cursorY++
	}
	buffer.cursorX = 0
}

func (buffer *Buffer) MovePosition(x int16, y int16) {

	var toX uint16
	var toY uint16

	if int16(buffer.cursorX)+x < 0 {
		toX = 0
	} else {
		toX = uint16(int16(buffer.cursorX) + x)
	}

	if int16(buffer.cursorY)+y < 0 {
		toY = 0
	} else {
		toY = uint16(int16(buffer.cursorY) + y)
	}

	buffer.SetPosition(toX, toY)
}

func (buffer *Buffer) ShowCursor() {

}

func (buffer *Buffer) HideCursor() {

}

func (buffer *Buffer) SetCursorBlink(enabled bool) {

}

func (buffer *Buffer) SetPosition(col uint16, line uint16) {
	defer buffer.emitDisplayChange()
	if col >= buffer.ViewWidth() {
		col = buffer.ViewWidth() - 1
		//logrus.Errorf("Cannot set cursor position: column %d is outside of the current view width (%d columns)", col, buffer.ViewWidth())
	}
	if line >= buffer.ViewHeight() {
		line = buffer.ViewHeight() - 1
		//logrus.Errorf("Cannot set cursor position: line %d is outside of the current view height (%d lines)", line, buffer.ViewHeight())
	}
	buffer.cursorX = col
	buffer.cursorY = line
}

func (buffer *Buffer) GetVisibleLines() []Line {
	lines := []Line{}
	for i := buffer.Height() - int(buffer.ViewHeight()); i < buffer.Height(); i++ {
		if i >= 0 && i < len(buffer.lines) {
			lines = append(lines, buffer.lines[i])
		}
	}
	return lines
}

// tested to here

func (buffer *Buffer) Clear() {
	defer buffer.emitDisplayChange()
	for i := 0; i < int(buffer.ViewHeight()); i++ {
		buffer.lines = append(buffer.lines, newLine())
	}
	buffer.SetPosition(0, 0) // do we need to set position?
}

// creates if necessary
func (buffer *Buffer) getCurrentLine() *Line {

	if buffer.cursorY >= buffer.ViewHeight() {
		panic(fmt.Sprintf("cursor is outside of view: %d %d", buffer.cursorX, buffer.cursorY))
	}

	if len(buffer.lines) < int(buffer.ViewHeight()) {
		for int(buffer.cursorY) >= len(buffer.lines) {
			buffer.lines = append(buffer.lines, newLine())
		}
		return &buffer.lines[int(buffer.cursorY)]
	}

	if int(buffer.RawLine()) < len(buffer.lines) {
		return &buffer.lines[buffer.RawLine()]
	}

	panic(fmt.Sprintf("Failed to retrieve line for %d %d", buffer.cursorX, buffer.cursorY))
}

func (buffer *Buffer) EraseLine() {
	defer buffer.emitDisplayChange()
	line := buffer.getCurrentLine()
	line.cells = []Cell{}
}

func (buffer *Buffer) EraseLineToCursor() {
	defer buffer.emitDisplayChange()
	line := buffer.getCurrentLine()
	for i := 0; i <= int(buffer.cursorX); i++ {
		if i < len(line.cells) {
			line.cells[i].erase()
		}
	}
}

func (buffer *Buffer) EraseLineFromCursor() {
	defer buffer.emitDisplayChange()
	line := buffer.getCurrentLine()

	max := int(buffer.cursorX)
	if max > len(line.cells) {
		max = len(line.cells)
	}

	for c := int(buffer.cursorX); c < len(line.cells); c++ {
		line.cells[c].erase()
	}
}

func (buffer *Buffer) EraseDisplay() {
	defer buffer.emitDisplayChange()
	for i := uint16(0); i < (buffer.ViewHeight()); i++ {
		rawLine := buffer.convertViewLineToRawLine(i)
		if int(rawLine) < len(buffer.lines) {
			buffer.lines[int(rawLine)].cells = []Cell{}
		}
	}
}

func (buffer *Buffer) EraseDisplayFromCursor() {
	defer buffer.emitDisplayChange()
	line := buffer.getCurrentLine()

	line.cells = line.cells[:buffer.cursorX]
	for i := buffer.cursorY + 1; i < buffer.ViewHeight(); i++ {
		rawLine := buffer.convertViewLineToRawLine(i)
		if int(rawLine) < len(buffer.lines) {
			buffer.lines[int(rawLine)].cells = []Cell{}
		}
	}
}

func (buffer *Buffer) EraseDisplayToCursor() {
	defer buffer.emitDisplayChange()
	line := buffer.getCurrentLine()

	for i := 0; i < int(buffer.cursorX); i++ {
		line.cells[i].erase()
	}
	for i := uint16(0); i < buffer.cursorY; i++ {
		rawLine := buffer.convertViewLineToRawLine(i)
		if int(rawLine) < len(buffer.lines) {
			buffer.lines[int(rawLine)].cells = []Cell{}
		}
	}
}

func (buffer *Buffer) ResizeView(width uint16, height uint16) {
	defer buffer.emitDisplayChange()
	buffer.viewWidth = width
	buffer.viewHeight = height

	// @todo wrap/unwrap
}
