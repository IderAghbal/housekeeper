package parser

type (
	// CreateRoleStmt represents CREATE ROLE statements
	// Syntax: CREATE ROLE [IF NOT EXISTS | OR REPLACE] name [ON CLUSTER cluster] [SETTINGS ...];
	CreateRoleStmt struct {
		LeadingCommentField
		OrReplace   bool          `parser:"'CREATE' (@'OR' 'REPLACE')? 'ROLE'"`
		IfNotExists bool          `parser:"@('IF' 'NOT' 'EXISTS')?"`
		Name        string        `parser:"@(Ident | BacktickIdent)"`
		OnCluster   *string       `parser:"('ON' 'CLUSTER' @(Ident | BacktickIdent))?"`
		Settings    *RoleSettings `parser:"@@?"`
		TrailingCommentField
		Semicolon bool `parser:"';'"`
	}

	// AlterRoleStmt represents ALTER ROLE statements
	// Syntax: ALTER ROLE [IF EXISTS] name [ON CLUSTER cluster] [RENAME TO new_name] [SETTINGS ...];
	AlterRoleStmt struct {
		LeadingCommentField
		IfExists  bool          `parser:"'ALTER' 'ROLE' @('IF' 'EXISTS')?"`
		Name      string        `parser:"@(Ident | BacktickIdent)"`
		OnCluster *string       `parser:"('ON' 'CLUSTER' @(Ident | BacktickIdent))?"`
		RenameTo  *string       `parser:"('RENAME' 'TO' @(Ident | BacktickIdent))?"`
		Settings  *RoleSettings `parser:"@@?"`
		TrailingCommentField
		Semicolon bool `parser:"';'"`
	}

	// DropRoleStmt represents DROP ROLE statements
	// Syntax: DROP ROLE [IF EXISTS] name [,...] [ON CLUSTER cluster];
	DropRoleStmt struct {
		LeadingCommentField
		IfExists  bool     `parser:"'DROP' 'ROLE' @('IF' 'EXISTS')?"`
		Names     []string `parser:"@(Ident | BacktickIdent) (',' @(Ident | BacktickIdent))*"`
		OnCluster *string  `parser:"('ON' 'CLUSTER' @(Ident | BacktickIdent))?"`
		TrailingCommentField
		Semicolon bool `parser:"';'"`
	}

	// SetRoleStmt represents SET ROLE statements for session management
	// Syntax: SET ROLE {DEFAULT | NONE | ALL | ALL EXCEPT name [,...] | name [,...]};
	SetRoleStmt struct {
		Default   bool      `parser:"'SET' 'ROLE' (@'DEFAULT'"`
		None      bool      `parser:"| @'NONE'"`
		All       bool      `parser:"| @'ALL'"`
		AllExcept *RoleList `parser:"| 'ALL' 'EXCEPT' @@"`
		Roles     *RoleList `parser:"| @@)"`
		Semicolon bool      `parser:"';'"`
	}

	// SetDefaultRoleStmt represents SET DEFAULT ROLE statements
	// Syntax: SET DEFAULT ROLE {NONE | ALL | name [,...] | ALL EXCEPT name [,...]} TO user [,...];
	SetDefaultRoleStmt struct {
		None      bool      `parser:"'SET' 'DEFAULT' 'ROLE' (@'NONE'"`
		All       bool      `parser:"| @'ALL'"`
		Roles     *RoleList `parser:"| @@"`
		AllExcept *RoleList `parser:"| 'ALL' 'EXCEPT' @@)"`
		ToUsers   []string  `parser:"'TO' @(Ident | BacktickIdent) (',' @(Ident | BacktickIdent))*"`
		Semicolon bool      `parser:"';'"`
	}

	// GrantStmt represents GRANT statements.
	//
	// ClickHouse syntax (https://clickhouse.com/docs/en/sql-reference/statements/grant):
	//
	//   GRANT [ON CLUSTER cluster_name]
	//         privilege[(column_name [,...])] [,...]
	//         ON {db.table[*]|db.*|*.*|table[*]|*}
	//         TO {user | role | CURRENT_USER} [,...]
	//         [WITH GRANT OPTION] [WITH REPLACE OPTION] [WITH ADMIN OPTION]
	//
	// ClickHouse accepts ON CLUSTER in two positions: the canonical
	// leading slot (immediately after GRANT, before privileges) and a
	// trailing slot (after the grantee list, before WITH). The
	// previous grammar placed it between Privileges and the ON target,
	// a third position ClickHouse rejects with a syntax error — so
	// any DDL the old format emitted was invalid against a real
	// cluster.
	//
	// This grammar parses either accepted position into LeadCluster
	// or TrailCluster; call OnCluster() to get the effective value
	// regardless of where it appeared.
	//
	// The cluster name may be an identifier, a back-ticked identifier,
	// or a quoted string (the latter so `'{cluster}'` macros parse).
	GrantStmt struct {
		LeadingCommentField
		LeadCluster  *string        `parser:"'GRANT' ('ON' 'CLUSTER' @(Ident | BacktickIdent | String))?"`
		Privileges   *PrivilegeList `parser:"@@"`
		On           *GrantTarget   `parser:"('ON' @@)?"`
		To           *GranteeList   `parser:"'TO' @@"`
		TrailCluster *string        `parser:"('ON' 'CLUSTER' @(Ident | BacktickIdent | String))?"`
		WithGrant    bool           `parser:"@('WITH' 'GRANT' 'OPTION')?"`
		WithReplace  bool           `parser:"@('WITH' 'REPLACE' 'OPTION')?"`
		WithAdmin    bool           `parser:"@('WITH' 'ADMIN' 'OPTION')?"`
		TrailingCommentField
		Semicolon bool `parser:"';'"`
	}

	// RevokeStmt represents REVOKE statements.
	//
	// ClickHouse syntax: REVOKE [ON CLUSTER cluster_name]
	//   [GRANT OPTION FOR | ADMIN OPTION FOR] privilege[,...] ON ...
	//   FROM {user|role|CURRENT_USER}[,...]
	//
	// As with GrantStmt, ON CLUSTER is accepted at the canonical
	// leading position or trailing after the grantee list.
	RevokeStmt struct {
		LeadingCommentField
		LeadCluster  *string        `parser:"'REVOKE' ('ON' 'CLUSTER' @(Ident | BacktickIdent | String))?"`
		GrantOption  bool           `parser:"(@'GRANT' 'OPTION' 'FOR'"`
		AdminOption  bool           `parser:"| @'ADMIN' 'OPTION' 'FOR')?"`
		Privileges   *PrivilegeList `parser:"@@"`
		On           *GrantTarget   `parser:"('ON' @@)?"`
		From         *GranteeList   `parser:"'FROM' @@"`
		TrailCluster *string        `parser:"('ON' 'CLUSTER' @(Ident | BacktickIdent | String))?"`
		TrailingCommentField
		Semicolon bool `parser:"';'"`
	}

	// RoleSettings represents SETTINGS clause for roles
	RoleSettings struct {
		Settings []*RoleSetting `parser:"'SETTINGS' @@ (',' @@)*"`
	}

	// RoleSetting represents a single role setting
	RoleSetting struct {
		Name  string  `parser:"@(Ident | BacktickIdent)"`
		Value *string `parser:"('=' @(Number | String | Ident | BacktickIdent))?"`
	}

	// RoleList represents a list of role names
	RoleList struct {
		Names []string `parser:"@(Ident | BacktickIdent) (',' @(Ident | BacktickIdent))*"`
	}

	// PrivilegeList represents a list of privileges or roles in GRANT/REVOKE
	PrivilegeList struct {
		Items []*PrivilegeItem `parser:"@@ (',' @@)*"`
	}

	// PrivilegeItem represents a single privilege or role
	PrivilegeItem struct {
		// This is simplified - in reality, privileges can be complex expressions
		// We'll parse them as identifiers and let ClickHouse validate
		All     bool     `parser:"@'ALL'"`
		Name    string   `parser:"| @(Ident | BacktickIdent)"`
		Columns []string `parser:"('(' @(Ident | BacktickIdent) (',' @(Ident | BacktickIdent))* ')')?"`
	}

	// GrantTarget represents the target of a GRANT/REVOKE (database.table or *.*)
	GrantTarget struct {
		Star1    *string `parser:"( @'*'"`
		Star2    *string `parser:"  '.' @'*'"`
		Database *string `parser:"| @(Ident | BacktickIdent)"`
		Table    *string `parser:"  '.' @(Ident | BacktickIdent | '*'))"`
	}

	// GranteeList represents the list of users/roles receiving privileges
	GranteeList struct {
		Items []*Grantee `parser:"@@ (',' @@)*"`
	}

	// Grantee represents a user or role receiving privileges
	Grantee struct {
		Name      string `parser:"@(Ident | BacktickIdent)"`
		IsCurrent bool   `parser:"| @'CURRENT_USER'"`
	}
)

// OnCluster returns the effective ON CLUSTER target regardless of
// whether it was parsed in the leading or trailing position. nil
// when neither was specified.
func (g *GrantStmt) OnCluster() *string {
	if g.LeadCluster != nil {
		return g.LeadCluster
	}
	return g.TrailCluster
}

// OnCluster returns the effective ON CLUSTER target regardless of
// whether it was parsed in the leading or trailing position. nil
// when neither was specified.
func (r *RevokeStmt) OnCluster() *string {
	if r.LeadCluster != nil {
		return r.LeadCluster
	}
	return r.TrailCluster
}
