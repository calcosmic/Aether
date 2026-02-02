# TODO: Phase Completion Improvements

## Todo 1: Next Steps Recommendation

**Description**: At the end of each stage (phase completion), the system should recommend to the user which commands to run next.

**Where to implement**: Phase completion handlers (end of /ant:build, coordinator completion)

**Requirements**:
- When a phase completes, display clear next steps
- Show available commands with brief descriptions
- Prioritize next logical action (usually next phase)
- Include alternative options (review, status, etc.)

**Example output after phase completion**:
```
╔══════════════════════════════════════════════════════════════╗
║  Phase X Complete! 🐜                                      ║
╚══════════════════════════════════════════════════════════════╝

Recommended Next Steps:

1. /ant:build {NEXT_PHASE} - Continue to Phase {NEXT_PHASE}
2. /ant:phase X - Review completed phase details
3. /ant:status - View colony status
4. /ant:focus <area> - Set focus for next phase (optional)

💡 Tip: Clear context (/clear) before starting next phase for fresh context window.
```

**Implementation locations**:
- .claude/commands/ant/build.md - Update phase completion section
- .aether/workers/*.md - Update completion reporting in caste prompts

---

## Todo 2: Context Handoff Reminder

**Description**: At the end of each stage, ensure proper context handoff and remind the user to clear context before beginning a new stage.

**Where to implement**: Phase completion handlers, /cds:pause-work workflow

**Requirements**:
- Create .continue-here.md file automatically at phase completion
- Remind user to clear context after phase completion
- Provide clear command to resume work
- Ensure state is persisted before recommending context clear

**Example output at phase completion**:
```
╔══════════════════════════════════════════════════════════════╗
║  Phase X Complete! Context Handoff Created                   ║
╚══════════════════════════════════════════════════════════════╝

✅ Phase state persisted
✅ Handoff file created: .planning/phases/XX-name/.continue-here.md

RECOMMENDED WORKFLOW:

1. Review completed work:
   /ant:phase X

2. Clear context for fresh start:
   /clear

3. Resume in new context:
   /cds:resume-work

This ensures:
- ✓ Clean context window for next phase
- ✓ Full state restoration
- ✓ No context rot from previous work
```

**Implementation locations**:
- .claude/commands/ant/build.md - Add context handoff section
- .planning/phases/XX-name/.continue-here.md - Auto-generate at completion
- Update phase completion workflow to include handoff

---

## Priority

Both todos are HIGH priority for user experience:
- Todo 1 improves clarity of next actions
- Todo 2 prevents context rot and improves session continuity

## Dependencies

- Todo 2 should be implemented first (context handoff foundation)
- Todo 1 builds on Todo 2 by adding recommendations to handoff

## Success Criteria

### Todo 1:
- [ ] Every phase completion shows next steps
- [ ] Commands are actionable and clear
- [ ] Recommendations are contextually relevant
- [ ] Tips for best practices included

### Todo 2:
- [ ] Handoff file created at every phase completion
- [ ] User is reminded to clear context
- [ ] Resume command is clearly provided
- [ ] State is fully persisted before context clear recommendation
