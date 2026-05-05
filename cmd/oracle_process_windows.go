//go:build windows

package cmd

import (
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func terminateOracleProcessTree(rootPID int) ([]int, error) {
	if rootPID <= 0 || !oracleProcessExists(rootPID) {
		return nil, nil
	}
	if err := exec.Command("taskkill", "/PID", strconv.Itoa(rootPID), "/T", "/F").Run(); err != nil {
		return nil, fmt.Errorf("terminate oracle process tree %d: %w", rootPID, err)
	}
	return []int{rootPID}, nil
}

func oracleProcessExists(pid int) bool {
	if pid <= 0 {
		return false
	}
	output, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/FO", "CSV", "/NH").Output()
	if err != nil {
		return false
	}
	text := strings.TrimSpace(string(output))
	if text == "" || strings.Contains(strings.ToLower(text), "no tasks are running") {
		return false
	}
	records, err := csv.NewReader(strings.NewReader(text)).ReadAll()
	if err != nil {
		return false
	}
	want := strconv.Itoa(pid)
	for _, record := range records {
		if len(record) > 1 && strings.TrimSpace(record[1]) == want {
			return true
		}
	}
	return false
}
