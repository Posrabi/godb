package src

import (
	"errors"
)

type columnType uint

const (
	TextType columnType = iota
	IntType
	BoolType
)

type Cell interface {
	AsText() string
	AsInt() int32
	AsBool() bool
}

type Results struct {
	Columns []ResultsColumn
	Rows    [][]Cell
}

type ResultsColumn struct {
	Type columnType
	Name string
}

var (
	TableDoesNotExists        = errors.New("Table does not exist")
	TableAlreadyExists        = errors.New("Table already exists")
	ColumnDoesNotExist        = errors.New("Column does not exist")
	InvalidSelectItem         = errors.New("Select item is not valid")
	InvalidDatatype           = errors.New("Invalid datatype")
	MissingValues             = errors.New("Missing values")
	InvalidCell               = errors.New("Cell is invalid")
	InvalidOperands           = errors.New("Operands are invalid")
	IndexAlreadyExists        = errors.New("Index already exists")
	PrimaryKeyAlreadyExists   = errors.New("Primary key already exists")
	ViolatesNonNullConstraint = errors.New("Violates non-null constraint")
	ViolatesUniqueConstraint  = errors.New("Violates unique constraint")
)

type Backend interface {
	CreateTable(*CreateTableStatement) error
	Insert(*InsertStatement) error
	Select(*SelectStatement) (*Results, error)
}
