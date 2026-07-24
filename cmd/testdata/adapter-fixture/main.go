package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type workerClaims struct {
	AntName       string         `json:"ant_name"`
	Caste         string         `json:"caste"`
	TaskID        string         `json:"task_id"`
	Status        string         `json:"status"`
	Summary       string         `json:"summary"`
	FilesCreated  []string       `json:"files_created"`
	FilesModified []string       `json:"files_modified"`
	TestsWritten  []string       `json:"tests_written"`
	Artifacts     map[string]any `json:"artifacts"`
	ToolCount     int            `json:"tool_count"`
	Blockers      []string       `json:"blockers"`
	Spawns        []string       `json:"spawns"`
	Handoff       workerHandoff  `json:"handoff"`
}

type workerHandoff struct {
	ChangedFiles           []string `json:"changed_files"`
	CommandsRun            []string `json:"commands_run"`
	VerificationStatus     string   `json:"verification_status"`
	KnownFailures          []string `json:"known_failures"`
	OpenDecisions          []string `json:"open_decisions"`
	Assumptions            []string `json:"assumptions"`
	NextWorkerInstructions []string `json:"next_worker_instructions"`
	DoNotRepeat            []string `json:"do_not_repeat"`
	Freshness              string   `json:"freshness"`
}

func main() {
	prompt, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	outputPath := argumentValue(os.Args[1:], "--output-last-message")
	if outputPath == "" {
		fmt.Println(`{"type":"fixture.preflight","status":"ok"}`)
		return
	}

	mode := strings.ToLower(strings.TrimSpace(os.Getenv("AETHER_TEST_ADAPTER_MODE")))
	if mode == "" {
		mode = "success"
	}
	worker := promptField(string(prompt), "Worker")
	caste := strings.ToLower(promptField(string(prompt), "Caste"))
	taskID := promptTaskID(string(prompt))
	logInvocation(mode, worker, caste)

	switch mode {
	case "timeout":
		time.Sleep(10 * time.Second)
		return
	case "partial-timeout":
		mustWrite("interrupted-worker-output.txt", []byte("worker wrote this before interruption\n"))
		time.Sleep(30 * time.Second)
		return
	case "crash":
		fmt.Fprintln(os.Stderr, "fixture adapter crash")
		os.Exit(17)
	case "malformed":
		mustWrite(outputPath, []byte("not worker claims"))
		return
	case "partial":
		partial := "partial-" + safeName(firstNonEmpty(worker, caste, "worker")) + ".txt"
		mustWrite(partial, []byte("partial worker output\n"))
		fmt.Fprintln(os.Stderr, "fixture adapter stopped after a partial write")
		os.Exit(18)
	case "success", "no-op":
	default:
		fmt.Fprintf(os.Stderr, "unknown fixture mode %q\n", mode)
		os.Exit(2)
	}

	created := []string{}
	if mode == "success" && caste == "builder" {
		mustWrite("app.txt", []byte("created by deterministic adapter\n"))
		created = append(created, "app.txt")
		if taskID != "" {
			artifact := "journey-" + safeName(taskID) + ".txt"
			mustWrite(artifact, []byte("completed "+taskID+" through deterministic provider dispatch\n"))
			created = append(created, artifact)
		}
	}
	if mode == "success" && caste == "oracle" {
		responsePath := promptLineValue(string(prompt), "Response File:")
		questionID := promptJSONStringValue(string(prompt), "question_id")
		if responsePath != "" && questionID != "" {
			response := map[string]any{
				"question_id": questionID,
				"status":      "answered",
				"confidence":  100,
				"summary":     "The deterministic provider produced repository-scoped research evidence.",
				"findings": []map[string]any{
					{
						"text": "The journey fixture is a compiled CLI repository with an isolated provider process.",
						"evidence": []map[string]string{{
							"title":    "compiled acceptance fixture",
							"location": "cmd/blackbox_harness_test.go",
							"type":     "codebase",
						}},
					},
				},
				"gaps":           []string{},
				"contradictions": []string{},
				"recommendation": "Use evidence-bound plans and deterministic finalizers for the acceptance journey.",
			}
			data, marshalErr := json.Marshal(response)
			if marshalErr != nil {
				fmt.Fprintln(os.Stderr, marshalErr)
				os.Exit(2)
			}
			mustWrite(responsePath, data)
			created = append(created, filepath.ToSlash(responsePath))
		}
	}
	claims := workerClaims{
		AntName:       worker,
		Caste:         caste,
		TaskID:        taskID,
		Status:        "completed",
		Summary:       fmt.Sprintf("fixture %s completed for %s", mode, firstNonEmpty(worker, caste, "worker")),
		FilesCreated:  created,
		FilesModified: []string{},
		TestsWritten:  []string{},
		Artifacts:     map[string]any{},
		ToolCount:     1,
		Blockers:      []string{},
		Spawns:        []string{},
		Handoff: workerHandoff{
			ChangedFiles:           append([]string{}, created...),
			CommandsRun:            []string{"deterministic-adapter " + mode},
			VerificationStatus:     "not_run",
			KnownFailures:          []string{},
			OpenDecisions:          []string{},
			Assumptions:            []string{},
			NextWorkerInstructions: []string{},
			DoNotRepeat:            []string{},
			Freshness:              time.Now().UTC().Format(time.RFC3339),
		},
	}
	data, err := json.Marshal(claims)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	mustWrite(outputPath, data)
}

func argumentValue(args []string, name string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return ""
}

func promptField(prompt, name string) string {
	prefix := "- " + name + ":"
	scanner := bufio.NewScanner(strings.NewReader(prompt))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func promptTaskID(prompt string) string {
	scanner := bufio.NewScanner(strings.NewReader(prompt))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# Task ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# Task "))
		}
	}
	return ""
}

func promptLineValue(prompt, prefix string) string {
	scanner := bufio.NewScanner(strings.NewReader(prompt))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func promptJSONStringValue(prompt, key string) string {
	prefix := `"` + key + `": "`
	scanner := bufio.NewScanner(strings.NewReader(prompt))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		value := strings.TrimPrefix(line, prefix)
		value = strings.TrimSuffix(value, ",")
		value = strings.TrimSuffix(value, `"`)
		return strings.TrimSpace(value)
	}
	return ""
}

func logInvocation(mode, worker, caste string) {
	path := strings.TrimSpace(os.Getenv("AETHER_TEST_ADAPTER_LOG"))
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	entry, _ := json.Marshal(map[string]any{
		"mode":   mode,
		"worker": worker,
		"caste":  caste,
		"pid":    os.Getpid(),
	})
	_, _ = file.Write(append(entry, '\n'))
}

func mustWrite(path string, data []byte) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func safeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
