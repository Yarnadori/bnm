package main

import "strings"

// closestMatch returns the candidate closest to input by edit distance,
// or "" when nothing is close enough to be a plausible typo.
func closestMatch(input string, candidates []string) string {
	threshold := 2
	if len(input) <= 4 {
		threshold = 1
	}

	best := ""
	bestDist := threshold + 1
	for _, c := range candidates {
		d := editDistance(strings.ToLower(input), strings.ToLower(c))
		if d < bestDist {
			best = c
			bestDist = d
		}
	}
	if bestDist == 0 {
		// Exact (case-insensitive) match is not a typo suggestion
		return ""
	}
	return best
}

// editDistance computes the optimal string alignment distance between two
// strings: Levenshtein plus adjacent transpositions (so "dve" -> "dev" is 1).
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev2 := make([]int, len(rb)+1)
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, min(curr[j-1]+1, prev[j-1]+cost))
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] {
				curr[j] = min(curr[j], prev2[j-2]+1)
			}
		}
		prev2, prev, curr = prev, curr, prev2
	}
	return prev[len(rb)]
}
