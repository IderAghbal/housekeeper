package schema

import (
	"testing"

	"github.com/pseudomuto/housekeeper/pkg/parser"
	"github.com/stretchr/testify/require"
)

func TestCompoundPrivilegeGrantsReadBackEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		declared string
		// server is the grant as the dump writes it from system.grants.
		server string
	}{
		{
			name:     "table",
			declared: `GRANT ON CLUSTER '{cluster}' ALTER UPDATE ON drive.files TO sweeper;`,
			server:   "GRANT ON CLUSTER '{cluster}' ALTER UPDATE ON `drive`.`files` TO `sweeper`;",
		},
		{
			name:     "columns",
			declared: `GRANT ALTER UPDATE(state) ON drive.files TO sweeper;`,
			server:   "GRANT ALTER UPDATE(`state`) ON `drive`.`files` TO `sweeper`;",
		},
		{
			name:     "every database",
			declared: `GRANT SYSTEM FLUSH LOGS ON *.* TO operator;`,
			server:   "GRANT SYSTEM FLUSH LOGS ON *.* TO `operator`;",
		},
		{
			name:     "spaced declaration",
			declared: "GRANT ALTER   DELETE ON drive.versions TO sweeper;",
			server:   "GRANT ALTER DELETE ON `drive`.`versions` TO `sweeper`;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			declared, err := parser.ParseString(tt.declared)
			require.NoError(t, err)
			server, err := parser.ParseString(tt.server)
			require.NoError(t, err)

			require.Empty(t, compareRoles(server, declared))
		})
	}
}

func TestAMissingCompoundPrivilegeIsGrantedWhole(t *testing.T) {
	t.Parallel()

	declared, err := parser.ParseString(`GRANT ON CLUSTER '{cluster}' ALTER UPDATE ON drive.files TO sweeper;`)
	require.NoError(t, err)
	server, err := parser.ParseString("GRANT ON CLUSTER '{cluster}' ALTER DELETE ON `drive`.`files` TO `sweeper`;")
	require.NoError(t, err)

	diffs := compareRoles(server, declared)
	require.Len(t, diffs, 2)
	sql := map[string]string{}
	for _, d := range diffs {
		sql[d.Type] = d.UpSQL
	}
	require.Equal(t, "GRANT ON CLUSTER '{cluster}' ALTER UPDATE ON drive.files TO sweeper;", sql[string(RoleDiffGrant)])
	require.Equal(t, "REVOKE ON CLUSTER '{cluster}' ALTER DELETE ON drive.files FROM sweeper;", sql[string(RoleDiffRevoke)])
}
