package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/store"
)

func answerModeLabel(mode string) string {
	if store.NormalizeAnswerMode(mode) == store.AnswerModeWrite {
		return i18n.T(i18n.AnswerModeWrite)
	}
	return i18n.T(i18n.AnswerModeChoice)
}

func sessionModeLabel(mode string) string {
	if store.IsInfiniteReviewMode(mode) {
		return i18n.T(i18n.StartModeReviewPractice)
	}
	if store.IsReviewMode(mode) {
		return i18n.T(i18n.StartModeReview)
	}
	return i18n.T(i18n.StartModeLearn)
}

func nextHintIndices(word string, shown []int, hintCount int) []int {
	runes := []rune(word)
	if len(runes) == 0 {
		return nil
	}

	seen := make(map[int]struct{}, len(shown))
	for _, index := range shown {
		if index >= 0 && index < len(runes) {
			seen[index] = struct{}{}
		}
	}

	if hintCount == 0 {
		seen[0] = struct{}{}
		if len(runes) >= 5 {
			seen[len(runes)-1] = struct{}{}
		}
		return sortedHintIndices(seen)
	}

	for _, index := range centerOutIndices(len(runes)) {
		if _, ok := seen[index]; ok {
			continue
		}
		seen[index] = struct{}{}
		break
	}

	return sortedHintIndices(seen)
}

func renderSlots(word string, shown []int) string {
	runes := []rune(word)
	if len(runes) == 0 {
		return ""
	}

	visible := make(map[int]struct{}, len(shown))
	for _, index := range shown {
		if index >= 0 && index < len(runes) {
			visible[index] = struct{}{}
		}
	}

	parts := make([]string, 0, len(runes))
	for i, letter := range runes {
		if _, ok := visible[i]; ok {
			parts = append(parts, string(letter))
		} else {
			parts = append(parts, "_")
		}
	}
	return strings.Join(parts, " ")
}

func renderSpacedInput(input string) string {
	runes := []rune(input)
	if len(runes) == 0 {
		return ""
	}

	parts := make([]string, 0, len(runes))
	for _, letter := range runes {
		parts = append(parts, string(letter))
	}
	return strings.Join(parts, " ")
}

func centerOutIndices(length int) []int {
	if length <= 0 {
		return nil
	}

	indices := make([]int, 0, length)
	if length%2 == 1 {
		center := length / 2
		indices = append(indices, center)
		for step := 1; len(indices) < length; step++ {
			left := center - step
			right := center + step
			if left >= 0 {
				indices = append(indices, left)
			}
			if right < length {
				indices = append(indices, right)
			}
		}
		return indices
	}

	leftCenter := (length / 2) - 1
	rightCenter := length / 2
	indices = append(indices, leftCenter, rightCenter)
	for step := 1; len(indices) < length; step++ {
		left := leftCenter - step
		right := rightCenter + step
		if left >= 0 {
			indices = append(indices, left)
		}
		if right < length {
			indices = append(indices, right)
		}
	}
	return indices
}

func sortedHintIndices(seen map[int]struct{}) []int {
	indices := make([]int, 0, len(seen))
	for index := range seen {
		indices = append(indices, index)
	}
	sort.Ints(indices)
	return indices
}

func formatHintCount(count int) string {
	if count <= 0 {
		return i18n.T(i18n.QuizHintNone)
	}
	return fmt.Sprintf("%d", count)
}
