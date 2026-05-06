package cmd

func renderWorkerReadCacheDiscipline() string {
	return `## Read Cache Discipline

- Read each target or evidence file once for understanding. Do not re-read the same unchanged file for confidence.
- If a read tool says "File unchanged since last read" or tells you to refer to earlier content, treat the earlier content as authoritative and continue from it.
- If you need one small detail, use rg/Grep or a narrow targeted read for the symbol or line range. Do not loop full-file reads.
- After editing, verify with tests, build output, git diff, or a targeted read of the changed area.
- If you still cannot proceed after two attempts because necessary context is missing, return blocked with the missing context. Do not keep reading.
`
}
