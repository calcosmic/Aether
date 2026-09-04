package cmd

import "testing"

func TestClassicCommandParity(t *testing.T) {
	manifest := loadClassicPublicCommandParity(t)
	assertClassicPublicCommandParity(t, manifest)
}
