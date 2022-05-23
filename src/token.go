package src

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
)

type tokenKind uint

const (
	KeywordKind tokenKind = iota
	SymbolKind
	IdentifierKind
	StringKind
	NumericKind
	BoolKind
)

type token struct {
	value string
	kind  tokenKind
	loc   location
}

var (
	trueToken  = token{kind: BoolKind, value: "true"}
	falseToken = token{kind: BoolKind, value: "false"}

	trueMemoryCell  = trueToken.literalToMemoryCell()
	falseMemoryCell = falseToken.literalToMemoryCell()
)

func (t *token) literalToMemoryCell() memoryCell {
	if t.kind == NumericKind {
		buf := new(bytes.Buffer)
		i, err := strconv.Atoi(t.value)
		if err != nil {
			fmt.Printf("Corrupted data [%s]: %s\n", t.value, err)
		}

		err = binary.Write(buf, binary.BigEndian, int32(i))
		if err != nil {
			fmt.Printf("Corrupted data [%s]: %s\n", string(buf.Bytes()), err)
		}
		return memoryCell(buf.Bytes())
	}

	if t.kind == StringKind {
		return memoryCell(t.value)
	}

	if t.kind == BoolKind {
		if t.value == "true" {
			return memoryCell([]byte{1})
		}
		return memoryCell(nil)
	}

	return nil
}

func (t *token) equals(other *token) bool {
	return t.value == other.value && t.kind == other.kind
}

func (t *token) bindingPower() uint {
	switch t.kind {
	case KeywordKind:
		switch keyword(t.value) {
		case And:
			fallthrough
		case Or:
			return 1
		}
	case SymbolKind:
		switch symbol(t.value) {
		case Equal:
			fallthrough
		case XEqual:
			fallthrough
		case Concat:
			fallthrough
		case Plus:
			return 3
		}
	}

	return 0
}
