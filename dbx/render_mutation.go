//revive:disable:file-length-limit Mutation rendering helpers are kept together to preserve related SQL behavior.

package dbx

import (
	"errors"
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx"
	"github.com/DaiYuANg/arcgo/dbx/dialect"
)

func (q *InsertQuery) Build(d dialect.Dialect) (BoundQuery, error) {
	if q == nil {
		return BoundQuery{}, errors.New("dbx: insert query is nil")
	}
	rows := normalizedInsertRows(q)
	if err := validateInsertQuery(q, rows); err != nil {
		return BoundQuery{}, err
	}

	state := &renderState{dialect: d, args: make([]any, 0, rows.RowCount()*4)}
	state.writeString(insertStatementPrefix(d, q))
	if err := renderInsertBody(state, q, rows); err != nil {
		return BoundQuery{}, err
	}
	if err := renderUpsert(state, q); err != nil {
		return BoundQuery{}, err
	}
	if err := renderReturning(state, q.ReturningItems); err != nil {
		return BoundQuery{}, err
	}
	if err := state.err(); err != nil {
		return BoundQuery{}, err
	}
	return state.BoundQuery(), nil
}

func validateInsertQuery(q *InsertQuery, rows collectionx.Grid[Assignment]) error {
	switch {
	case q.Into.Name() == "":
		return errors.New("dbx: insert query requires target table")
	case rows.RowCount() == 0 && q.Source == nil:
		return errors.New("dbx: insert query requires values or source query")
	case rows.RowCount() > 0 && q.Source != nil:
		return errors.New("dbx: insert query cannot combine values and source query")
	case q.Source != nil && q.TargetColumns.Len() == 0:
		return errors.New("dbx: insert-select requires target columns")
	default:
		return nil
	}
}

func insertStatementPrefix(d dialect.Dialect, q *InsertQuery) string {
	features := dialectFeatures(d)
	if features.InsertIgnoreForUpsertNothing && q.Upsert != nil && q.Upsert.DoNothing {
		return "INSERT IGNORE INTO "
	}
	return "INSERT INTO "
}

func renderInsertBody(state *renderState, q *InsertQuery, rows collectionx.Grid[Assignment]) error {
	state.renderTable(q.Into)
	columns, err := resolveInsertColumns(q, rows)
	if err != nil {
		return err
	}
	if err := renderInsertColumns(state, columns); err != nil {
		return err
	}
	return renderInsertSourceOrValues(state, q, columns, rows)
}

func renderInsertColumns(state *renderState, columns collectionx.List[ColumnMeta]) error {
	if columns.Len() == 0 {
		return nil
	}
	state.writeString(" (")
	columns.Range(func(index int, column ColumnMeta) bool {
		if index > 0 {
			state.writeString(", ")
		}
		state.writeQuotedIdent(column.Name)
		return true
	})
	state.writeByte(')')
	return nil
}

func renderInsertSourceOrValues(state *renderState, q *InsertQuery, columns collectionx.List[ColumnMeta], rows collectionx.Grid[Assignment]) error {
	if q.Source != nil {
		state.writeByte(' ')
		return renderSelectQuery(state, q.Source)
	}
	return renderInsertValues(state, columns, rows)
}

func renderInsertValues(state *renderState, columns collectionx.List[ColumnMeta], rows collectionx.Grid[Assignment]) error {
	orderedRows, err := orderInsertRows(columns, rows)
	if err != nil {
		return err
	}
	state.writeString(" VALUES ")
	var renderErr error
	orderedRows.Range(func(rowIndex int, row []Assignment) bool {
		if rowIndex > 0 {
			state.writeString(", ")
		}
		if err := renderInsertValueRow(state, row); err != nil {
			renderErr = err
			return false
		}
		return true
	})
	return renderErr
}

func renderInsertValueRow(state *renderState, row []Assignment) error {
	state.writeByte('(')
	for colIndex, assignment := range row {
		renderer, ok := assignment.(insertAssignmentRenderer)
		if !ok {
			return fmt.Errorf("dbx: unsupported insert assignment %T", assignment)
		}
		if colIndex > 0 {
			state.writeString(", ")
		}
		if err := renderer.renderAssignmentValue(state); err != nil {
			return err
		}
	}
	state.writeByte(')')
	return nil
}

func (q *UpdateQuery) Build(d dialect.Dialect) (BoundQuery, error) {
	if q == nil {
		return BoundQuery{}, errors.New("dbx: update query is nil")
	}
	if err := validateUpdateQuery(q); err != nil {
		return BoundQuery{}, err
	}

	state := &renderState{dialect: d, args: make([]any, 0, q.Assignments.Len())}
	state.writeString("UPDATE ")
	state.renderTable(q.Table)
	state.writeString(" SET ")
	if err := renderUpdateAssignments(state, q.Assignments); err != nil {
		return BoundQuery{}, err
	}
	if err := renderOptionalWhere(state, q.WhereExp); err != nil {
		return BoundQuery{}, err
	}
	if err := renderReturning(state, q.ReturningItems); err != nil {
		return BoundQuery{}, err
	}
	if err := state.err(); err != nil {
		return BoundQuery{}, err
	}
	return state.BoundQuery(), nil
}

func validateUpdateQuery(q *UpdateQuery) error {
	switch {
	case q.Table.Name() == "":
		return errors.New("dbx: update query requires target table")
	case q.Assignments.Len() == 0:
		return errors.New("dbx: update query requires assignments")
	default:
		return nil
	}
}

func renderUpdateAssignments(state *renderState, assignments collectionx.List[Assignment]) error {
	var renderErr error
	assignments.Range(func(index int, assignment Assignment) bool {
		if index > 0 {
			state.writeString(", ")
		}
		if err := renderAssignment(state, assignment); err != nil {
			renderErr = err
			return false
		}
		return true
	})
	return renderErr
}

func renderOptionalWhere(state *renderState, predicate Predicate) error {
	if predicate == nil {
		return nil
	}
	state.writeString(" WHERE ")
	return renderPredicate(state, predicate)
}

func (q *DeleteQuery) Build(d dialect.Dialect) (BoundQuery, error) {
	if q == nil {
		return BoundQuery{}, errors.New("dbx: delete query is nil")
	}
	if q.From.Name() == "" {
		return BoundQuery{}, errors.New("dbx: delete query requires target table")
	}

	state := &renderState{dialect: d, args: make([]any, 0, 4)}
	state.writeString("DELETE FROM ")
	state.renderTable(q.From)
	if err := renderOptionalWhere(state, q.WhereExp); err != nil {
		return BoundQuery{}, err
	}
	if err := renderReturning(state, q.ReturningItems); err != nil {
		return BoundQuery{}, err
	}
	if err := state.err(); err != nil {
		return BoundQuery{}, err
	}
	return state.BoundQuery(), nil
}

func normalizedInsertRows(q *InsertQuery) collectionx.Grid[Assignment] {
	if q.Rows.RowCount() > 0 {
		return q.Rows
	}
	if q.Assignments.Len() > 0 {
		rows := collectionx.NewGridWithCapacity[Assignment](1)
		rows.AddRowList(q.Assignments)
		return rows
	}
	return nil
}

func resolveInsertColumns(q *InsertQuery, rows collectionx.Grid[Assignment]) (collectionx.List[ColumnMeta], error) {
	if q.TargetColumns.Len() > 0 {
		return resolveTargetColumns(q.TargetColumns)
	}
	row, ok := rows.FirstRowWhere(func(_ int, _ []Assignment) bool { return true }).Get()
	if !ok {
		return collectionx.NewList[ColumnMeta](), nil
	}
	return assignmentColumns(row)
}

func assignmentColumns(assignments []Assignment) (collectionx.List[ColumnMeta], error) {
	columns := collectionx.NewListWithCapacity[ColumnMeta](len(assignments))
	for _, assignment := range assignments {
		renderer, ok := assignment.(insertAssignmentRenderer)
		if !ok {
			return nil, fmt.Errorf("dbx: unsupported insert assignment %T", assignment)
		}
		columns.Add(renderer.assignmentColumn())
	}
	return columns, nil
}

func resolveTargetColumns(expressions collectionx.List[Expression]) (collectionx.List[ColumnMeta], error) {
	columns := collectionx.NewListWithCapacity[ColumnMeta](expressions.Len())
	var resolveErr error
	expressions.Range(func(_ int, expression Expression) bool {
		column, ok := expression.(columnAccessor)
		if !ok {
			resolveErr = fmt.Errorf("dbx: unsupported target column expression %T", expression)
			return false
		}
		columns.Add(column.columnRef())
		return true
	})
	if resolveErr != nil {
		return nil, resolveErr
	}
	return columns, nil
}

func orderInsertRows(columns collectionx.List[ColumnMeta], rows collectionx.Grid[Assignment]) (collectionx.Grid[Assignment], error) {
	orderedRows := collectionx.NewGridWithCapacity[Assignment](rows.RowCount())
	var orderErr error
	rows.Range(func(_ int, row []Assignment) bool {
		orderedRow, err := orderInsertRow(columns, row)
		if err != nil {
			orderErr = err
			return false
		}
		orderedRows.AddRowList(orderedRow)
		return true
	})
	if orderErr != nil {
		return nil, orderErr
	}
	return orderedRows, nil
}

func orderInsertRow(columns collectionx.List[ColumnMeta], row []Assignment) (collectionx.List[Assignment], error) {
	assignmentsByColumn := collectionx.NewMapWithCapacity[string, Assignment](len(row))
	for _, assignment := range row {
		renderer, ok := assignment.(insertAssignmentRenderer)
		if !ok {
			return nil, fmt.Errorf("dbx: unsupported insert assignment %T", assignment)
		}
		assignmentsByColumn.Set(renderer.assignmentColumn().Name, assignment)
	}

	orderedRow := collectionx.NewListWithCapacity[Assignment](columns.Len())
	var orderErr error
	columns.Range(func(_ int, column ColumnMeta) bool {
		assignment, ok := assignmentsByColumn.Get(column.Name)
		if !ok {
			orderErr = fmt.Errorf("dbx: missing value for insert column %s", column.Name)
			return false
		}
		orderedRow.Add(assignment)
		return true
	})
	if orderErr != nil {
		return nil, orderErr
	}
	return orderedRow, nil
}
