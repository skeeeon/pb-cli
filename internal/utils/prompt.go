package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Confirm prints prompt to stderr and reads a yes/no answer from stdin.
// It returns true only when the user answers "y" or "yes" (case-insensitive).
// Prompts go to stderr so they never contaminate piped stdout data.
func Confirm(prompt string) (bool, error) {
	fmt.Fprint(os.Stderr, prompt)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// ConfirmWord prints prompt to stderr and requires the user to type an exact
// word (case-sensitive) to confirm a dangerous operation. It returns true only
// when the typed response matches word exactly.
func ConfirmWord(prompt, word string) (bool, error) {
	fmt.Fprint(os.Stderr, prompt)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	return strings.TrimSpace(response) == word, nil
}
