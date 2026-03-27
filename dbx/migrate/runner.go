package migrate

// RunReport describes migrations applied by a runner operation.
type RunReport struct {
	Applied []AppliedRecord
}
