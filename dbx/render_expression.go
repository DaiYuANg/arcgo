package dbx

import (
	"errors"
	"fmt"
	"strings"
)

func (c Column[E, T]) renderOperand(state *renderState) (string, error) {
	meta := c.columnRef()
	var builder renderBuffer
	table := meta.Table
	if meta.Alias != "" {
		table = meta.Alias
	}
	builder.writeString(state.dialect.QuoteIdent(table))
	builder.writeByte('.')
	builder.writeString(state.dialect.QuoteIdent(meta.Name))
	return builder.String(), builder.Err("render column operand")
}

func (o valueOperand[T]) renderOperand(state *renderState) (string, error) {
	return state.bind(o.Value), nil
}

func (o columnOperand[T]) renderOperand(state *renderState) (string, error) {
	meta := o.Column.columnRef()
	var builder renderBuffer
	table := meta.Table
	if meta.Alias != "" {
		table = meta.Alias
	}
	builder.writeString(state.dialect.QuoteIdent(table))
	builder.writeByte('.')
	builder.writeString(state.dialect.QuoteIdent(meta.Name))
	return builder.String(), builder.Err("render column operand")
}

func (p comparisonPredicate) renderPredicate(state *renderState) error {
	left, err := p.Left.renderOperand(state)
	if err != nil {
		return err
	}
	state.writeString(left)
	if p.Op == OpIs || p.Op == OpIsNot {
		state.writeByte(' ')
		state.writeString(string(p.Op))
		state.writeString(" NULL")
		return nil
	}
	operand, err := renderOperandValue(state, p.Right)
	if err != nil {
		return err
	}
	state.writeByte(' ')
	state.writeString(string(p.Op))
	state.writeByte(' ')
	state.writeString(operand)
	return nil
}

func (p logicalPredicate) renderPredicate(state *renderState) error {
	if p.Predicates.Len() == 0 {
		return errors.New("dbx: logical predicate requires nested predicates")
	}
	state.writeByte('(')
	var renderErr error
	p.Predicates.Range(func(index int, predicate Predicate) bool {
		if index > 0 {
			state.writeByte(' ')
			state.writeString(string(p.Op))
			state.writeByte(' ')
		}
		if err := renderPredicate(state, predicate); err != nil {
			renderErr = err
			return false
		}
		return true
	})
	if renderErr != nil {
		return renderErr
	}
	state.writeByte(')')
	return nil
}

func (p notPredicate) renderPredicate(state *renderState) error {
	if p.Predicate == nil {
		return errors.New("dbx: NOT predicate requires nested predicate")
	}
	state.writeString("NOT (")
	if err := renderPredicate(state, p.Predicate); err != nil {
		return err
	}
	state.writeByte(')')
	return nil
}

func (p existsPredicate) renderPredicate(state *renderState) error {
	if p.Query == nil {
		return errors.New("dbx: EXISTS predicate requires subquery")
	}
	state.writeString("EXISTS (")
	if err := renderSelectStatement(state, p.Query); err != nil {
		return err
	}
	state.writeByte(')')
	return nil
}

func (a columnAssignment[E, T]) assignmentColumn() ColumnMeta {
	return a.Column.columnRef()
}

func (a columnAssignment[E, T]) renderAssignment(state *renderState) error {
	state.writeQuotedIdent(a.Column.Name())
	state.writeString(" = ")
	operand, err := renderOperandValue(state, a.Value)
	if err != nil {
		return err
	}
	state.writeString(operand)
	return nil
}

func (a columnAssignment[E, T]) renderAssignmentValue(state *renderState) error {
	operand, err := renderOperandValue(state, a.Value)
	if err != nil {
		return err
	}
	state.writeString(operand)
	return nil
}

func (o columnOrder[E, T]) renderOrder(state *renderState) error {
	state.renderColumn(o.Column.columnRef())
	if o.Descending {
		state.writeString(" DESC")
		return nil
	}
	state.writeString(" ASC")
	return nil
}

func (o expressionOrder) renderOrder(state *renderState) error {
	operand, err := o.Expr.renderOperand(state)
	if err != nil {
		return err
	}
	state.writeString(operand)
	if o.Descending {
		state.writeString(" DESC")
		return nil
	}
	state.writeString(" ASC")
	return nil
}

func (a Aggregate[T]) renderOperand(state *renderState) (string, error) {
	var builder renderBuffer
	builder.writeString(string(a.Function))
	builder.writeByte('(')
	if a.Distinct {
		builder.writeString("DISTINCT ")
	}
	if a.star {
		builder.writeByte('*')
	} else {
		if a.Expr == nil {
			return "", fmt.Errorf("dbx: aggregate %s requires expression", a.Function)
		}
		operand, err := a.Expr.renderOperand(state)
		if err != nil {
			return "", err
		}
		builder.writeString(operand)
	}
	builder.writeByte(')')
	return builder.String(), builder.Err("render aggregate operand")
}

func (a Aggregate[T]) renderSelectItem(state *renderState) error {
	operand, err := a.renderOperand(state)
	if err != nil {
		return err
	}
	state.writeString(operand)
	return nil
}

func (c CaseExpression[T]) renderOperand(state *renderState) (string, error) {
	if c.Branches.Len() == 0 {
		return "", errors.New("dbx: CASE expression requires at least one WHEN branch")
	}

	var builder renderBuffer
	builder.writeString("CASE")
	var renderErr error
	c.Branches.Range(func(_ int, branch caseWhenBranch) bool {
		if err := renderCaseBranch(&builder, state, branch); err != nil {
			renderErr = err
			return false
		}
		return true
	})
	if renderErr != nil {
		return "", renderErr
	}
	if err := renderCaseElse(&builder, state, c.Else); err != nil {
		return "", err
	}
	builder.writeString(" END")
	return builder.String(), builder.Err("render case operand")
}

func renderCaseBranch(builder *renderBuffer, state *renderState, branch caseWhenBranch) error {
	if branch.Predicate == nil {
		return errors.New("dbx: CASE branch requires predicate")
	}
	builder.writeString(" WHEN ")
	predicateSQL, err := renderPredicateValue(state, branch.Predicate)
	if err != nil {
		return err
	}
	builder.writeString(predicateSQL)
	builder.writeString(" THEN ")
	valueSQL, err := renderOperandValue(state, branch.Value)
	if err != nil {
		return err
	}
	builder.writeString(valueSQL)
	return nil
}

func renderCaseElse(builder *renderBuffer, state *renderState, value any) error {
	if value == nil {
		return nil
	}
	builder.writeString(" ELSE ")
	elseSQL, err := renderOperandValue(state, value)
	if err != nil {
		return err
	}
	builder.writeString(elseSQL)
	return nil
}

func (c CaseExpression[T]) renderSelectItem(state *renderState) error {
	operand, err := c.renderOperand(state)
	if err != nil {
		return err
	}
	state.writeString(operand)
	return nil
}

func (o excludedColumnOperand[T]) renderOperand(state *renderState) (string, error) {
	f := dialectFeatures(state.dialect)
	quoted := state.dialect.QuoteIdent(o.Column.Name)
	switch f.ExcludedRefStyle {
	case "excluded":
		return "EXCLUDED." + quoted, nil
	case "values":
		return "VALUES(" + quoted + ")", nil
	default:
		return "", fmt.Errorf("dbx: excluded assignment is not supported for dialect %s", state.dialect.Name())
	}
}

func (a aliasedSelectItem) renderSelectItem(state *renderState) error {
	if a.Item == nil {
		return errors.New("dbx: aliased select item requires value")
	}
	if err := renderAliasedItemValue(state, a.Item); err != nil {
		return err
	}
	if strings.TrimSpace(a.Alias) == "" {
		return nil
	}
	state.writeString(" AS ")
	state.writeQuotedIdent(strings.TrimSpace(a.Alias))
	return nil
}

func renderAliasedItemValue(state *renderState, item any) error {
	switch renderer := item.(type) {
	case selectItemRenderer:
		return renderer.renderSelectItem(state)
	case operandRenderer:
		operand, err := renderer.renderOperand(state)
		if err != nil {
			return err
		}
		state.writeString(operand)
		return nil
	default:
		return fmt.Errorf("dbx: unsupported aliased select item %T", item)
	}
}
