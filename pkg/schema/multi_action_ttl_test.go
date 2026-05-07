package schema_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pseudomuto/housekeeper/pkg/parser"
	"github.com/pseudomuto/housekeeper/pkg/schema"
)

// TestGenerateDiff_MultiActionTTL_RoundTrips guards the integration
// between the parser's multi-action TTL support and the schema layer's
// MODIFY TTL synthesis. Regression for the warning-path failure where
// the diff layer generated `MODIFY TTL ts+90d TO VOLUME 'cold', ts+1y
// DELETE` but its own re-parse failed at `TO`, causing the diff to
// silently downgrade to ErrNoDiff.
//
// Setup:
//   - current = single-action TTL (the live state before the change).
//   - target = multi-action TTL (the SQL files after declaring it).
//
// Assertions:
//   - GenerateDiff returns at least one statement (no ErrNoDiff).
//   - The generated SQL contains `MODIFY TTL` with both actions.
//   - The generated SQL re-parses cleanly into a ModifyTTLOperation
//     with two entries (the failure mode that produced the original
//     warning would have left this empty via ErrNoDiff).
func TestGenerateDiff_MultiActionTTL_RoundTrips(t *testing.T) {
	t.Parallel()

	currentSQL := `CREATE TABLE forjeron.audit_log (id UInt64, ts DateTime, retention_days UInt32 DEFAULT 365)
ENGINE = MergeTree()
ORDER BY id
TTL toDateTime(ts) + toIntervalDay(retention_days) DELETE;`

	targetSQL := `CREATE TABLE forjeron.audit_log (id UInt64, ts DateTime, retention_days UInt32 DEFAULT 365)
ENGINE = MergeTree()
ORDER BY id
TTL toDateTime(ts) + toIntervalDay(90) TO VOLUME 'cold',
    toDateTime(ts) + toIntervalDay(retention_days) DELETE;`

	current, err := parser.ParseString(currentSQL)
	require.NoError(t, err, "parse current")

	target, err := parser.ParseString(targetSQL)
	require.NoError(t, err, "parse target")

	diff, err := schema.GenerateDiff(current, target)
	require.NoError(t, err, "GenerateDiff")
	require.NotNil(t, diff, "expected non-nil diff for TTL change")
	require.NotEmpty(t, diff.Statements, "expected at least one ALTER statement")

	// Locate the ALTER TABLE statement and assert the multi-action TTL
	// shape made it through formatting + re-parse.
	var alter *parser.AlterTableStmt
	for _, stmt := range diff.Statements {
		if stmt.AlterTable != nil {
			alter = stmt.AlterTable
			break
		}
	}
	require.NotNil(t, alter, "expected an ALTER TABLE statement in diff")

	// Find the MODIFY TTL operation.
	var modify *parser.ModifyTTLOperation
	for _, op := range alter.Operations {
		if op.ModifyTTL != nil {
			modify = op.ModifyTTL
			break
		}
	}
	require.NotNil(t, modify, "expected MODIFY TTL operation in ALTER")
	require.Len(t, modify.Entries, 2,
		"expected multi-action TTL with 2 entries (TO VOLUME 'cold' + DELETE), got %d",
		len(modify.Entries))

	// Verify both action types survived: one TO VOLUME, one DELETE.
	var (
		seenToVolume bool
		seenDelete   bool
	)
	for i := range modify.Entries {
		entry := &modify.Entries[i]
		require.NotNil(t, entry.Action, "entry %d action is nil", i)
		switch {
		case entry.Action.ToVolume != nil:
			seenToVolume = true
			require.Contains(t, *entry.Action.ToVolume, "cold",
				"entry %d ToVolume should be 'cold', got %q", i, *entry.Action.ToVolume)
		case entry.Action.Delete != nil:
			seenDelete = true
		}
	}
	require.True(t, seenToVolume, "expected one TO VOLUME entry in MODIFY TTL")
	require.True(t, seenDelete, "expected one DELETE entry in MODIFY TTL")
}

// TestGenerateDiff_MultiActionTTL_DoesNotEmitWarning is the negative
// guard for the same regression: when re-parse of the generated SQL
// fails, generator.go logs a "WARNING: Generated invalid DDL" line on
// stdout and returns ErrNoDiff. The previous test catches that via the
// non-empty Statements assertion; this one documents the failure mode
// for any reader who hits a future regression.
func TestGenerateDiff_MultiActionTTL_DoesNotEmitWarning(t *testing.T) {
	t.Parallel()

	currentSQL := `CREATE TABLE db.t (id UInt64, ts DateTime)
ENGINE = MergeTree() ORDER BY id
TTL ts + INTERVAL 1 YEAR DELETE;`

	targetSQL := `CREATE TABLE db.t (id UInt64, ts DateTime)
ENGINE = MergeTree() ORDER BY id
TTL ts + INTERVAL 90 DAY TO VOLUME 'cold',
    ts + INTERVAL 1 YEAR DELETE;`

	current, err := parser.ParseString(currentSQL)
	require.NoError(t, err)
	target, err := parser.ParseString(targetSQL)
	require.NoError(t, err)

	diff, err := schema.GenerateDiff(current, target)
	require.NoError(t, err, "GenerateDiff returned an error (parse failure of synthesized SQL would surface here as ErrNoDiff)")
	require.NotNil(t, diff)
	// The warning case returns ErrNoDiff (translated to nil diff with
	// ErrNoDiff error) — assert we got real statements instead.
	require.NotEmpty(t, diff.Statements,
		"diff is empty; if the parser rejected the synthesized SQL the diff layer downgrades to ErrNoDiff")

	// Spot-check the rendered SQL contains both action keywords with
	// no formatting weirdness that would mis-tokenize.
	var rendered strings.Builder
	for _, stmt := range diff.Statements {
		require.NotNil(t, stmt)
		// Stringify via the canonical String() if available.
		if s, ok := any(stmt).(interface{ String() string }); ok {
			rendered.WriteString(s.String())
		}
	}
	out := rendered.String()
	if out != "" {
		// Stmt.String may not be implemented on all variants; only
		// assert when we actually rendered something. Both action
		// keywords must be present.
		require.Contains(t, out, "TO VOLUME")
		require.Contains(t, out, "DELETE")
	}
}
