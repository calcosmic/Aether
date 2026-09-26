package cmd

import (
	"regexp"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// The owner page is the plain-English view of one specification revision:
// what you'll get, what's included, what you decided, the rules, what is
// off-limits, what happens if something goes wrong, and how you'll check it.
// It is presentation only. It never shows item IDs, fingerprints or the
// revision-impact bookkeeping, strips the machine-generated lead-ins the
// draft builders add (buildSettledDiscussDraftRequest in discuss.go and the
// /ant-go derived draft in go_default_path.go), and says each distinct
// sentence once. The full listing with every stable ID stays available
// through `aether spec --detail`, and the JSON result is untouched.

// specOwnerLeadIn is one machine-generated opening phrase. When Standalone is
// set and the text after the lead-in has already been said elsewhere on the
// page, the standalone sentence is shown instead, so the meaning of the
// lead-in (for example "keep the last working version") is not lost when its
// subject is a repeat.
type specOwnerLeadIn struct {
	Prefix     string
	Standalone string
}

const specOwnerRecoverySentence = "If this goes wrong, the last working version is kept and you are told before anything carries on."

var specOwnerLeadIns = []specOwnerLeadIn{
	{Prefix: "Deliver the accepted goal: "},
	{Prefix: "Use the accepted implementation context: "},
	{Prefix: "Work outside these accepted constraints is excluded: "},
	{Prefix: "The result must not violate these accepted constraints: "},
	{Prefix: "The result must not silently realize this known risk: "},
	{Prefix: "If the known risk occurs, preserve the last valid state and report it before continuing: ", Standalone: specOwnerRecoverySentence},
	{Prefix: "Implement only behavior needed to satisfy the accepted goal: "},
	{Prefix: "Implement only behavior needed to satisfy the requested job: "},
	{Prefix: "Settled behavior: "},
	{Prefix: "Settled scope boundary: "},
	{Prefix: "Owner-checkable acceptance: "},
	{Prefix: "Settled risk boundary: "},
	{Prefix: "If the settled risk boundary is crossed, preserve the last valid state and return to the owner: ", Standalone: specOwnerRecoverySentence},
	{Prefix: "The result must not violate this explicit owner constraint: "},
	{Prefix: "The result must not violate this active owner constraint: "},
	{Prefix: "Active owner constraint: "},
	{Prefix: "Required result: "},
	{Prefix: "The owner can verify that the delivered behavior satisfies the accepted goal: "},
	{Prefix: "The owner can verify that the delivered behavior satisfies the requested job: "},
}

// specOwnerDecisionMarker is how buildSettledDiscussDraftRequest joins an
// owner clarification's question and answer.
const specOwnerDecisionMarker = " — Owner decision: "

type specOwnerPageSection struct {
	Kind  string
	Title string
	Items []string
}

var specOwnerSentenceBoundaryRe = regexp.MustCompile(`[.?!]\s+`)

// specOwnerSentenceKey normalises a sentence for the say-it-once rule:
// case, runs of whitespace and closing punctuation are ignored.
func specOwnerSentenceKey(sentence string) string {
	key := strings.ToLower(strings.Join(strings.Fields(sentence), " "))
	return strings.TrimRight(key, ".?! ")
}

// specOwnerSentences splits text into sentences, keeping each sentence's own
// closing punctuation.
func specOwnerSentences(text string) []string {
	var sentences []string
	rest := strings.TrimSpace(text)
	for rest != "" {
		location := specOwnerSentenceBoundaryRe.FindStringIndex(rest)
		if location == nil {
			sentences = append(sentences, rest)
			break
		}
		sentences = append(sentences, strings.TrimSpace(rest[:location[0]+1]))
		rest = strings.TrimSpace(rest[location[1]:])
	}
	return sentences
}

// specOwnerStripLeadIn removes every machine lead-in from the front of text
// (they can stack, e.g. "Settled behavior: " before an owner decision) and
// reports the standalone sentence of the last lead-in that carried one.
func specOwnerStripLeadIn(text string) (string, string) {
	text = strings.TrimSpace(text)
	standalone := ""
	for {
		stripped := false
		for _, leadIn := range specOwnerLeadIns {
			if strings.HasPrefix(text, leadIn.Prefix) {
				text = strings.TrimSpace(strings.TrimPrefix(text, leadIn.Prefix))
				if leadIn.Standalone != "" {
					standalone = leadIn.Standalone
				}
				stripped = true
			}
		}
		if !stripped {
			return text, standalone
		}
	}
}

// specOwnerDecision reports an owner clarification as "question → answer".
func specOwnerDecision(text string) (string, bool) {
	question, answer, ok := strings.Cut(text, specOwnerDecisionMarker)
	if !ok {
		return "", false
	}
	question, answer = strings.TrimSpace(question), strings.TrimSpace(answer)
	if question == "" || answer == "" {
		return "", false
	}
	return question + " → " + answer, true
}

// specOwnerPageBuilder applies the say-it-once rule across the whole page.
type specOwnerPageBuilder struct {
	seen map[string]bool
}

// fresh returns only the sentences of text not already said on the page,
// marking them said.
func (b *specOwnerPageBuilder) fresh(text string) string {
	var kept []string
	for _, sentence := range specOwnerSentences(text) {
		key := specOwnerSentenceKey(sentence)
		if key == "" || b.seen[key] {
			continue
		}
		b.seen[key] = true
		kept = append(kept, sentence)
	}
	return strings.Join(kept, " ")
}

// item turns one stored description into what the owner reads, or "" when
// everything it says has already been said.
func (b *specOwnerPageBuilder) item(description string) string {
	text, standalone := specOwnerStripLeadIn(description)
	if decision, ok := specOwnerDecision(text); ok {
		key := specOwnerSentenceKey(decision)
		if b.seen[key] {
			return ""
		}
		b.seen[key] = true
		return decision
	}
	if kept := b.fresh(text); kept != "" {
		return kept
	}
	if standalone != "" {
		return b.fresh(standalone)
	}
	return ""
}

func (b *specOwnerPageBuilder) items(descriptions []string) []string {
	var out []string
	for _, description := range descriptions {
		if text := b.item(description); text != "" {
			out = append(out, text)
		}
	}
	return out
}

// specOwnerPageSections groups a revision's body into the owner's sections,
// in the owner's order, dropping empty ones. Owner decisions are claimed
// first so that their restatements elsewhere (as an included behavior or a
// hard constraint) are the copies that disappear, not the decision itself.
func specOwnerPageSections(body discussSpecificationBody) []specOwnerPageSection {
	builder := &specOwnerPageBuilder{seen: map[string]bool{}}

	var ownerDecisions, otherDecisions []string
	for _, decision := range body.BindingDecisions {
		text, _ := specOwnerStripLeadIn(decision.Description)
		if _, ok := specOwnerDecision(text); ok {
			ownerDecisions = append(ownerDecisions, decision.Description)
		} else {
			otherDecisions = append(otherDecisions, decision.Description)
		}
	}
	decided := builder.items(ownerDecisions)

	outcomes := make([]string, 0, len(body.Outcomes))
	for _, value := range body.Outcomes {
		outcomes = append(outcomes, value.Description)
	}
	included := make([]string, 0, len(body.IncludedBehaviors))
	for _, value := range body.IncludedBehaviors {
		included = append(included, value.Description)
	}
	rules := append([]string(nil), otherDecisions...)
	for _, value := range body.Requirements {
		rules = append(rules, value.Description)
	}
	offLimits := make([]string, 0, len(body.Exclusions)+len(body.NegativeExpectations))
	for _, value := range body.Exclusions {
		offLimits = append(offLimits, value.Description)
	}
	for _, value := range body.NegativeExpectations {
		offLimits = append(offLimits, value.Description)
	}
	recovery := make([]string, 0, len(body.RecoveryExpectations))
	for _, value := range body.RecoveryExpectations {
		recovery = append(recovery, value.Description)
	}
	checks := make([]string, 0, 2*len(body.AcceptanceChecks))
	for _, value := range body.AcceptanceChecks {
		checks = append(checks, value.Description, value.Verification)
	}

	// Each section is resolved in display order so the first place a
	// sentence can appear is the place it is kept.
	get := builder.items(outcomes)
	include := builder.items(included)
	rule := builder.items(rules)
	off := builder.items(offLimits)
	wrong := builder.items(recovery)
	check := builder.items(checks)

	candidates := []specOwnerPageSection{
		{Kind: "goal", Title: "What you'll get", Items: get},
		{Kind: "done", Title: "What's included", Items: include},
		{Kind: "decision", Title: "What you decided", Items: decided},
		{Kind: "requirement", Title: "Rules it must follow", Items: rule},
		{Kind: "avoid", Title: "Off-limits", Items: off},
		{Kind: "checkpoint", Title: "If something goes wrong", Items: wrong},
		{Kind: "evidence", Title: "How you'll check it", Items: check},
	}
	sections := make([]specOwnerPageSection, 0, len(candidates))
	for _, section := range candidates {
		if len(section.Items) > 0 {
			sections = append(sections, section)
		}
	}
	return sections
}

// specOwnerStatusLine says in plain words where this description stands.
func specOwnerStatusLine(status colony.SpecRevisionStatus) string {
	switch status {
	case colony.SpecStatusDraft:
		return "Draft — waiting for your approval. Nothing gets planned until you approve it."
	case colony.SpecStatusApproved:
		return "Approved — you signed this description off, so planning can go ahead."
	case colony.SpecStatusSuperseded:
		return "Replaced — a newer version of this description exists."
	default:
		return "Status: " + strings.ToLower(strings.TrimSpace(string(status)))
	}
}

// renderSpecOwnerPage renders the owner page body: the sections and one
// plain status line. Callers add the banner and the exact next command.
func renderSpecOwnerPage(body discussSpecificationBody, status colony.SpecRevisionStatus) string {
	var builder strings.Builder
	for _, section := range specOwnerPageSections(body) {
		builder.WriteString(renderStageMarker(section.Title))
		for _, item := range section.Items {
			builder.WriteString(voiceLine(section.Kind, item) + "\n")
		}
	}
	builder.WriteString(renderStageMarker("Where this stands"))
	builder.WriteString(voiceLine("status", specOwnerStatusLine(status)) + "\n")
	return builder.String()
}

// specCommandOwnerBody lifts the nine typed sections of a spec command result
// into the shared body shape the owner page reads.
func specCommandOwnerBody(result specCommandResult) discussSpecificationBody {
	return discussSpecificationBody{
		Outcomes:             result.Outcomes,
		IncludedBehaviors:    result.IncludedBehaviors,
		Exclusions:           result.Exclusions,
		BindingDecisions:     result.BindingDecisions,
		Requirements:         result.Requirements,
		AcceptanceChecks:     result.AcceptanceChecks,
		NegativeExpectations: result.NegativeExpectations,
		RecoveryExpectations: result.RecoveryExpectations,
		AffectedPublicPaths:  result.AffectedPublicPaths,
	}
}

// renderSpecCommandOwnerVisual is the default screen for `aether spec`
// inspection: the owner page, then the exact next command.
func renderSpecCommandOwnerVisual(result specCommandResult) string {
	var builder strings.Builder
	builder.WriteString(renderBanner("📜", "Specification"))
	builder.WriteString(voiceLine("artifact", "This is the description of what will be built. Read it, then approve it or say what to change.") + "\n")
	builder.WriteString(renderSpecOwnerPage(specCommandOwnerBody(result), result.Status))
	if result.Projection.Drifted {
		builder.WriteString(voiceLine("warning", "The readable copy saved in your project folder is out of date; the next step below rebuilds it without changing anything you approved.") + "\n")
	}
	if result.Status == colony.SpecStatusDraft && !result.Projection.Drifted {
		builder.WriteString(voiceLine("decision", "Happy with it? Run the command below to approve it. Want changes? Say what to change first.") + "\n")
	}
	builder.WriteString(voiceLine("files", "The full technical listing is one command away: `aether spec --detail`.") + "\n")
	builder.WriteString(renderNextUp(result.NextAction))
	return builder.String()
}

// renderSpecCommandVisualFor picks the screen: inspection shows the owner
// page unless the full listing was asked for; every change, approval and
// repair keeps the detailed listing with its revision impact.
func renderSpecCommandVisualFor(result specCommandResult, detail bool) string {
	if result.Operation == specCommandOperationInspect && !detail {
		return renderSpecCommandOwnerVisual(result)
	}
	return renderSpecCommandVisual(result)
}
