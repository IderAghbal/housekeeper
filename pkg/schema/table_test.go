package schema

import (
	"testing"

	"github.com/pseudomuto/housekeeper/pkg/parser"
	"github.com/stretchr/testify/require"
)

func TestEnginesEqual(t *testing.T) {
	tests := []struct {
		name     string
		target   *parser.TableEngine
		current  *parser.TableEngine
		expected bool
	}{
		{
			name: "ReplicatedMergeTree() target should equal ReplicatedMergeTree with params",
			target: &parser.TableEngine{
				Name:       "ReplicatedMergeTree",
				Parameters: []parser.EngineParameter{}, // No parameters
			},
			current: &parser.TableEngine{
				Name: "ReplicatedMergeTree",
				Parameters: []parser.EngineParameter{
					{String: stringPtr("'/clickhouse/tables/{uuid}/{shard}'")},
					{String: stringPtr("'{replica}'")},
				},
			},
			expected: true,
		},
		{
			name: "ReplicatedMergeTree with different explicit params should not be equal",
			target: &parser.TableEngine{
				Name: "ReplicatedMergeTree",
				Parameters: []parser.EngineParameter{
					{String: stringPtr("'/clickhouse/tables/new_path/{shard}'")},
					{String: stringPtr("'{replica}'")},
				},
			},
			current: &parser.TableEngine{
				Name: "ReplicatedMergeTree",
				Parameters: []parser.EngineParameter{
					{String: stringPtr("'/clickhouse/tables/old_path/{shard}'")},
					{String: stringPtr("'{replica}'")},
				},
			},
			expected: false,
		},
		{
			name: "MergeTree engines should use normal comparison",
			target: &parser.TableEngine{
				Name:       "MergeTree",
				Parameters: []parser.EngineParameter{},
			},
			current: &parser.TableEngine{
				Name:       "MergeTree",
				Parameters: []parser.EngineParameter{},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enginesEqual(tt.target, tt.current)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestTableInfoEqual(t *testing.T) {
	// Test ReplicatedMergeTree with different explicit parameters
	currentTable := &TableInfo{
		Name:     "events",
		Database: "",
		Engine: &parser.TableEngine{
			Name: "ReplicatedMergeTree",
			Parameters: []parser.EngineParameter{
				{String: stringPtr("'/clickhouse/tables/old_path/{shard}'")},
				{String: stringPtr("'{replica}'")},
			},
		},
		Columns: []ColumnInfo{
			{Name: "id", DataType: &parser.DataType{Simple: &parser.SimpleType{Name: "UInt64"}}},
			{Name: "data", DataType: &parser.DataType{Simple: &parser.SimpleType{Name: "String"}}},
		},
	}

	targetTable := &TableInfo{
		Name:     "events",
		Database: "",
		Engine: &parser.TableEngine{
			Name: "ReplicatedMergeTree",
			Parameters: []parser.EngineParameter{
				{String: stringPtr("'/clickhouse/tables/new_path/{shard}'")},
				{String: stringPtr("'{replica}'")},
			},
		},
		Columns: []ColumnInfo{
			{Name: "id", DataType: &parser.DataType{Simple: &parser.SimpleType{Name: "UInt64"}}},
			{Name: "data", DataType: &parser.DataType{Simple: &parser.SimpleType{Name: "String"}}},
		},
	}

	// These should NOT be equal due to different engine parameters
	result := currentTable.Equal(targetTable)
	require.False(t, result, "Tables with different ReplicatedMergeTree parameters should not be equal")
}

func TestInferSchemaCluster(t *testing.T) {
	cases := []struct {
		name     string
		tables   map[string]*TableInfo
		expected string
	}{
		{
			name: "macro cluster is authoritative",
			tables: map[string]*TableInfo{
				"events": {Cluster: "'{cluster}'"},
				"sales":  {Cluster: "'{cluster}'"},
			},
			expected: "'{cluster}'",
		},
		{
			name: "macro wins over plain names in the same map",
			tables: map[string]*TableInfo{
				"events": {Cluster: "default"},
				"sales":  {Cluster: "'{cluster}'"},
			},
			expected: "'{cluster}'",
		},
		{
			name: "unanimous plain name is returned",
			tables: map[string]*TableInfo{
				"events": {Cluster: "default"},
				"sales":  {Cluster: "default"},
			},
			expected: "default",
		},
		{
			name: "mixed plain names — no normalisation",
			tables: map[string]*TableInfo{
				"events": {Cluster: "cluster-a"},
				"sales":  {Cluster: "cluster-b"},
			},
			expected: "",
		},
		{
			name: "no clustered tables — no normalisation",
			tables: map[string]*TableInfo{
				"events": {Cluster: ""},
			},
			expected: "",
		},
		{
			name:     "empty map",
			tables:   map[string]*TableInfo{},
			expected: "",
		},
		{
			name: "shard macro is also accepted",
			tables: map[string]*TableInfo{
				"metrics": {Cluster: "'{shard}'"},
			},
			expected: "'{shard}'",
		},
		{
			name: "plain-quoted string without braces is NOT a macro",
			tables: map[string]*TableInfo{
				"events": {Cluster: "'plain-string'"},
			},
			expected: "'plain-string'", // treated as unanimous plain name
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := inferSchemaCluster(tc.tables)
			require.Equal(t, tc.expected, got)
		})
	}
}

// TestCompareTables_ClusterNormalisation verifies that ALTER/DROP/RENAME
// migrations use the target schema's cluster macro ('{cluster}') rather than
// the concrete cluster name that ClickHouse reports after macro expansion.
func TestCompareTables_ClusterNormalisation(t *testing.T) {
	// current = live DB state after applying migrations; ClickHouse has expanded
	// '{cluster}' → 'default' in SHOW CREATE TABLE output.
	current, err := parser.ParseString(`
		CREATE TABLE events ON CLUSTER default (
			id UInt64, name String
		) ENGINE = ReplicatedMergeTree() ORDER BY id;
	`)
	require.NoError(t, err)

	// target = desired schema with the portable macro syntax.
	target, err := parser.ParseString(`
		CREATE TABLE events ON CLUSTER '{cluster}' (
			id UInt64, name String, ts DateTime
		) ENGINE = ReplicatedMergeTree() ORDER BY id;
	`)
	require.NoError(t, err)

	diffs, err := compareTables(current, target)
	require.NoError(t, err)
	require.Len(t, diffs, 1)

	// The ALTER TABLE migration must use the macro, not the concrete name.
	require.Contains(t, diffs[0].UpSQL, "'{cluster}'",
		"ALTER TABLE UpSQL must use the macro cluster syntax")
	require.NotContains(t, diffs[0].UpSQL, "`default`",
		"ALTER TABLE UpSQL must not hard-code the concrete cluster name")
}

// TestCompareTables_DropClusterNormalisation verifies that DROP TABLE
// migrations generated for orphaned tables also use the target schema's cluster
// macro rather than the concrete cluster name from the live DB.
func TestCompareTables_DropClusterNormalisation(t *testing.T) {
	// current contains a table that no longer exists in the desired schema.
	current, err := parser.ParseString(`
		CREATE TABLE events ON CLUSTER default (
			id UInt64
		) ENGINE = ReplicatedMergeTree() ORDER BY id;
		CREATE TABLE orphan ON CLUSTER default (
			id UInt64
		) ENGINE = ReplicatedMergeTree() ORDER BY id;
	`)
	require.NoError(t, err)

	// target keeps only 'events', using the portable macro.
	target, err := parser.ParseString(`
		CREATE TABLE events ON CLUSTER '{cluster}' (
			id UInt64
		) ENGINE = ReplicatedMergeTree() ORDER BY id;
	`)
	require.NoError(t, err)

	diffs, err := compareTables(current, target)
	require.NoError(t, err)

	var dropDiff *TableDiff
	for _, d := range diffs {
		if d.Type == string(TableDiffDrop) {
			dropDiff = d
			break
		}
	}
	require.NotNil(t, dropDiff, "expected a DROP diff for orphan table")
	require.Contains(t, dropDiff.UpSQL, "'{cluster}'",
		"DROP TABLE UpSQL must use the macro cluster syntax")
	require.NotContains(t, dropDiff.UpSQL, "`default`",
		"DROP TABLE UpSQL must not hard-code the concrete cluster name")
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}

// dateTime64Type returns a parser DataType for DateTime64(6, 'UTC') for use in tests.
func dateTime64Type() *parser.DataType {
	return &parser.DataType{
		Simple: &parser.SimpleType{
			Name:       "DateTime64",
			Parameters: []parser.TypeParameter{{Number: stringPtr("6")}, {String: stringPtr("'UTC'")}},
		},
	}
}

func TestCompareColumns_RenameDetectedByPositionAndTypeOnly(t *testing.T) {
	// Same position + same type → rename detected even with zero name similarity
	// (e.g. min_event_received_at -> session_start_time)
	dateType := dateTime64Type()
	current := []ColumnInfo{
		{Name: "min_event_received_at", DataType: dateType},
		{Name: "max_event_received_at", DataType: dateType},
	}
	target := []ColumnInfo{
		{Name: "session_start_time", DataType: dateType},
		{Name: "session_end_time", DataType: dateType},
	}

	diffs := compareColumns(current, target)

	var renames []ColumnDiff
	for _, d := range diffs {
		if d.Type == ColumnDiffRename {
			renames = append(renames, d)
		}
	}
	require.Len(t, renames, 2, "expected 2 renames when position and type match")
	require.Equal(t, "min_event_received_at", renames[0].ColumnName)
	require.Equal(t, "session_start_time", renames[0].Target.Name)
	require.Equal(t, "max_event_received_at", renames[1].ColumnName)
	require.Equal(t, "session_end_time", renames[1].Target.Name)
}

func TestCompareColumns_RenameDetectedSamePositionSameTypeUnrelatedNames(t *testing.T) {
	// Same position, same type, completely unrelated names (foo, bar) → still rename
	uintType := &parser.DataType{Simple: &parser.SimpleType{Name: "UInt64"}}
	current := []ColumnInfo{
		{Name: "foo", DataType: uintType},
	}
	target := []ColumnInfo{
		{Name: "bar", DataType: uintType},
	}

	diffs := compareColumns(current, target)

	require.Len(t, diffs, 1)
	require.Equal(t, ColumnDiffRename, diffs[0].Type)
	require.Equal(t, "foo", diffs[0].ColumnName)
	require.Equal(t, "bar", diffs[0].Target.Name)
}

func TestCompareColumns_NoRenameWhenPositionDifferentSameType(t *testing.T) {
	// Different position, same type, low name similarity → ADD + DROP, not RENAME
	uintType := &parser.DataType{Simple: &parser.SimpleType{Name: "UInt64"}}
	current := []ColumnInfo{
		{Name: "a", DataType: uintType},
		{Name: "b", DataType: uintType},
	}
	target := []ColumnInfo{
		{Name: "x", DataType: uintType},
		{Name: "b", DataType: uintType},
		{Name: "y", DataType: uintType},
	}
	// Current: a(0), b(1). Target: x(0), b(1), y(2).
	// So we have DROP a, ADD x, ADD y. Position: a at 0, x at 0, y at 2.
	// Type match: a-x (both UInt64), a-y (both UInt64). Position: a-x match (0==0), a-y no (0!=2).
	// So a-x should match as rename (position+type). Then we have DROP nothing, ADD y. So one RENAME a->x, one ADD y.
	diffs := compareColumns(current, target)

	var renames []ColumnDiff
	var adds []ColumnDiff
	for _, d := range diffs {
		switch d.Type {
		case ColumnDiffRename:
			renames = append(renames, d)
		case ColumnDiffAdd:
			adds = append(adds, d)
		}
	}
	// a(0) and x(0) same type → rename. b exists in both. y(2) is new → ADD.
	require.Len(t, renames, 1, "only a->x should be rename (same position 0)")
	require.Equal(t, "a", renames[0].ColumnName)
	require.Equal(t, "x", renames[0].Target.Name)
	require.Len(t, adds, 1)
	require.Equal(t, "y", adds[0].ColumnName)
}

func TestCompareColumns_NoRenameWhenTypeDifferentSamePosition(t *testing.T) {
	// Same position, different type → ADD + DROP, not RENAME
	current := []ColumnInfo{
		{Name: "ts", DataType: &parser.DataType{Simple: &parser.SimpleType{Name: "DateTime"}}},
	}
	target := []ColumnInfo{
		{Name: "ts_utc", DataType: &parser.DataType{Simple: &parser.SimpleType{Name: "DateTime64", Parameters: []parser.TypeParameter{{Number: stringPtr("6")}}}}},
	}

	diffs := compareColumns(current, target)

	var renames []ColumnDiff
	for _, d := range diffs {
		if d.Type == ColumnDiffRename {
			renames = append(renames, d)
		}
	}
	require.Len(t, renames, 0, "different type should not produce rename")
	require.GreaterOrEqual(t, len(diffs), 1)
}
