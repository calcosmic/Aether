package cmd

// resolveReferenceSection returns a markdown section for skill references.
// Stub implementation -- another agent (99-01 wave 2) provides the full version.
func resolveReferenceSection(caste, task, workflow string) string {
	return ""
}

// appendMarkdownSections appends additional markdown sections to the base section.
// Stub implementation -- another agent (99-01 wave 2) provides the full version.
func appendMarkdownSections(base, additional string) string {
	if additional == "" {
		return base
	}
	if base == "" {
		return additional
	}
	return base + "\n" + additional
}
