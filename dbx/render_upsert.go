package dbx

import (
	"errors"
	"fmt"

	"github.com/DaiYuANg/arcgo/collectionx"
)

func renderUpsert(state *renderState, q *InsertQuery) error {
	if q.Upsert == nil {
		return nil
	}
	switch dialectFeatures(state.dialect).UpsertVariant {
	case "on_conflict":
		return renderUpsertOnConflict(state, q)
	case "on_duplicate_key":
		return renderUpsertOnDuplicateKey(state, q)
	default:
		return fmt.Errorf("dbx: upsert is not supported for dialect %s", state.dialect.Name())
	}
}

func renderUpsertOnConflict(state *renderState, q *InsertQuery) error {
	state.writeString(" ON CONFLICT")
	if err := renderUpsertTargets(state, q.Upsert.Targets); err != nil {
		return err
	}
	if q.Upsert.DoNothing {
		state.writeString(" DO NOTHING")
		return nil
	}
	if err := validateUpsertAssignments(q.Upsert); err != nil {
		return err
	}
	state.writeString(" DO UPDATE SET ")
	return renderUpsertAssignments(state, q.Upsert.Assignments)
}

func renderUpsertTargets(state *renderState, targets collectionx.List[Expression]) error {
	if targets.Len() == 0 {
		return nil
	}
	state.writeString(" (")
	var renderErr error
	targets.Range(func(index int, target Expression) bool {
		if index > 0 {
			state.writeString(", ")
		}
		if column, ok := target.(columnAccessor); ok {
			state.writeQuotedIdent(column.columnRef().Name)
			return true
		}
		operand, err := renderOperandValue(state, target)
		if err != nil {
			renderErr = err
			return false
		}
		state.writeString(operand)
		return true
	})
	if renderErr != nil {
		return renderErr
	}
	state.writeByte(')')
	return nil
}

func validateUpsertAssignments(upsert *UpsertClause) error {
	switch {
	case upsert.Assignments.Len() == 0:
		return errors.New("dbx: upsert update requires assignments")
	case upsert.Targets.Len() == 0:
		return errors.New("dbx: upsert update requires conflict targets")
	default:
		return nil
	}
}

func renderUpsertOnDuplicateKey(state *renderState, q *InsertQuery) error {
	if q.Upsert.DoNothing {
		return nil
	}
	if q.Upsert.Assignments.Len() == 0 {
		return errors.New("dbx: upsert update requires assignments")
	}
	state.writeString(" ON DUPLICATE KEY UPDATE ")
	return renderUpsertAssignments(state, q.Upsert.Assignments)
}

func renderUpsertAssignments(state *renderState, assignments collectionx.List[Assignment]) error {
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

func renderReturning(state *renderState, items collectionx.List[SelectItem]) error {
	if items.Len() == 0 {
		return nil
	}
	if !dialectFeatures(state.dialect).SupportsReturning {
		return fmt.Errorf("dbx: RETURNING is not supported for dialect %s", state.dialect.Name())
	}
	state.writeString(" RETURNING ")
	var renderErr error
	items.Range(func(index int, item SelectItem) bool {
		if index > 0 {
			state.writeString(", ")
		}
		if err := renderSelectItem(state, item); err != nil {
			renderErr = err
			return false
		}
		return true
	})
	return renderErr
}
