package dbx

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

type predicateRenderer interface {
	renderPredicate(*renderState) error
}

type assignmentRenderer interface {
	renderAssignment(*renderState) error
}

type insertAssignmentRenderer interface {
	assignmentRenderer
	renderAssignmentValue(*renderState) error
	assignmentColumn() ColumnMeta
}

type orderRenderer interface {
	renderOrder(*renderState) error
}

type operandRenderer interface {
	renderOperand(*renderState) (string, error)
}

type selectItemRenderer interface {
	renderSelectItem(*renderState) error
}

type renderState struct {
	dialect  dialect.Dialect
	buf      strings.Builder
	args     []any
	writeErr error
}

func (s *renderState) writeString(text string) {
	if s.writeErr != nil {
		return
	}

	_, s.writeErr = s.buf.WriteString(text)
}

func (s *renderState) writeByte(value byte) {
	if s.writeErr != nil {
		return
	}

	s.writeErr = s.buf.WriteByte(value)
}

func (s *renderState) err() error {
	return wrapDBError("write rendered SQL", s.writeErr)
}

type renderBuffer struct {
	buf strings.Builder
	err error
}

func (b *renderBuffer) writeString(text string) {
	if b.err != nil {
		return
	}

	_, b.err = b.buf.WriteString(text)
}

func (b *renderBuffer) writeByte(value byte) {
	if b.err != nil {
		return
	}

	b.err = b.buf.WriteByte(value)
}

func (b *renderBuffer) String() string {
	return b.buf.String()
}

func (b *renderBuffer) Err(op string) error {
	return wrapDBError(op, b.err)
}

func (s *renderState) bind(value any) string {
	s.args = append(s.args, value)
	return s.dialect.BindVar(len(s.args))
}

func (s *renderState) writeQuotedIdent(name string) {
	s.writeString(s.dialect.QuoteIdent(name))
}

func (s *renderState) writeQualifiedIdent(table, column string) {
	if table != "" {
		s.writeQuotedIdent(table)
		s.writeByte('.')
	}
	s.writeQuotedIdent(column)
}

func (s *renderState) renderColumn(meta ColumnMeta) {
	table := meta.Table
	if meta.Alias != "" {
		table = meta.Alias
	}
	s.writeQualifiedIdent(table, meta.Name)
}

func (s *renderState) renderTable(table Table) {
	s.writeQuotedIdent(table.Name())
	if alias := table.Alias(); alias != "" && alias != table.Name() {
		s.writeString(" AS ")
		s.writeQuotedIdent(alias)
	}
}

func (s *renderState) BoundQuery() BoundQuery {
	return BoundQuery{SQL: s.buf.String(), Args: slices.Clone(s.args)}
}

func renderSelectItem(state *renderState, item SelectItem) error {
	if renderer, ok := item.(selectItemRenderer); ok {
		return renderer.renderSelectItem(state)
	}
	column, ok := item.(columnAccessor)
	if !ok {
		return fmt.Errorf("dbx: unsupported select item %T", item)
	}
	state.renderColumn(column.columnRef())
	return nil
}

func renderPredicate(state *renderState, predicate Predicate) error {
	renderer, ok := predicate.(predicateRenderer)
	if !ok {
		return fmt.Errorf("dbx: unsupported predicate %T", predicate)
	}
	return renderer.renderPredicate(state)
}

func renderAssignment(state *renderState, assignment Assignment) error {
	renderer, ok := assignment.(assignmentRenderer)
	if !ok {
		return fmt.Errorf("dbx: unsupported assignment %T", assignment)
	}
	return renderer.renderAssignment(state)
}

func renderOrder(state *renderState, order Order) error {
	renderer, ok := order.(orderRenderer)
	if !ok {
		return fmt.Errorf("dbx: unsupported order %T", order)
	}
	return renderer.renderOrder(state)
}

func renderPredicateValue(state *renderState, predicate Predicate) (string, error) {
	original := state.buf
	var builder strings.Builder
	state.buf = builder
	if err := renderPredicate(state, predicate); err != nil {
		state.buf = original
		return "", err
	}
	rendered := state.buf.String()
	state.buf = original
	return rendered, nil
}

func renderOperandValue(state *renderState, value any) (string, error) {
	if renderer, ok := value.(operandRenderer); ok {
		return renderer.renderOperand(state)
	}
	if values, ok := value.([]any); ok {
		return renderAnySliceOperand(state, values)
	}
	return state.bind(value), nil
}

func renderAnySliceOperand(state *renderState, values []any) (string, error) {
	if len(values) == 0 {
		return "", errors.New("dbx: IN operand cannot be empty")
	}
	var builder renderBuffer
	builder.writeByte('(')
	for i, value := range values {
		if i > 0 {
			builder.writeString(", ")
		}
		builder.writeString(state.bind(value))
	}
	builder.writeByte(')')
	return builder.String(), builder.Err("render slice operand")
}

func dialectFeatures(d dialect.Dialect) dialect.QueryFeatures {
	if p, ok := d.(dialect.QueryFeaturesProvider); ok {
		return p.QueryFeatures()
	}
	return dialect.DefaultQueryFeatures(d.Name())
}
