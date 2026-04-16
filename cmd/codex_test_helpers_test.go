package cmd

import "fmt"

func validCodexAgentTOML(name, role string) []byte {
	if role == "" {
		role = name
	}
	return []byte(fmt.Sprintf(`name = %q
description = %q
nickname_candidates = [%q, %q]
developer_instructions = """
Use Codex CLI tools to work within the %s role.
Keep edits scoped and report verification results.
"""
`, name, fmt.Sprintf("%s test agent", name), role, name, role))
}
