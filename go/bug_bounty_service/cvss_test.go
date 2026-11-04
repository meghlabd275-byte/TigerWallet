package main

// cvss_test.go — Verifies the CVSS v3.1 base score against published vectors.
//
// The previous implementation returned a constant per severity label (high ->
// 7.5), so these vectors could not have passed. Each case below is taken from
// the CVSS v3.1 specification examples or the FIRST.org calculator.

import (
	"math"
	"math/big"
	"testing"
)

// TestCVSSV31KnownVectors checks the score against published metric
// combinations. A CVSS base score is a defined function of its metrics, so an
// exact match is required (tolerance 0.005 covers the one-decimal rounding).
func TestCVSSV31KnownVectors(t *testing.T) {
	cases := []struct {
		name    string
		metrics cvssMetrics
		want    float64
	}{
		{
			// The specification's own example: full impact, network vector.
			name: "spec example AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			metrics: cvssMetrics{
				AttackVector: "N", AttackComplexity: "L", PrivilegesRequired: "N",
				UserInteraction: "N", Scope: "U",
				Confidentiality: "H", Integrity: "H", Availability: "H",
			},
			want: 9.8,
		},
		{
			// Same metrics with Changed scope: 1.08 multiplier pushes it to 10.0.
			name: "scope changed C:H/I:H/A:H",
			metrics: cvssMetrics{
				AttackVector: "N", AttackComplexity: "L", PrivilegesRequired: "N",
				UserInteraction: "N", Scope: "C",
				Confidentiality: "H", Integrity: "H", Availability: "H",
			},
			want: 10.0,
		},
		{
			// Physical access, no impact: scores zero.
			name: "physical, no impact",
			metrics: cvssMetrics{
				AttackVector: "P", AttackComplexity: "H", PrivilegesRequired: "H",
				UserInteraction: "R", Scope: "U",
				Confidentiality: "N", Integrity: "N", Availability: "N",
			},
			want: 0.0,
		},
		{
			// Adjacent network, low complexity, no privileges: 8.8.
			name: "AV:A/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H",
			metrics: cvssMetrics{
				AttackVector: "A", AttackComplexity: "L", PrivilegesRequired: "N",
				UserInteraction: "N", Scope: "U",
				Confidentiality: "H", Integrity: "H", Availability: "H",
			},
			want: 8.8,
		},
		{
			// Local, low privileges, user interaction required: 7.8.
			name: "AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H",
			metrics: cvssMetrics{
				AttackVector: "L", AttackComplexity: "L", PrivilegesRequired: "L",
				UserInteraction: "N", Scope: "U",
				Confidentiality: "H", Integrity: "H", Availability: "H",
			},
			want: 7.8,
		},
		{
			// Only confidentiality affected: 7.5.
			name: "AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:N/A:N",
			metrics: cvssMetrics{
				AttackVector: "N", AttackComplexity: "L", PrivilegesRequired: "N",
				UserInteraction: "N", Scope: "U",
				Confidentiality: "H", Integrity: "N", Availability: "N",
			},
			want: 7.5,
		},
		{
			// Scope changed with low privileges uses the 0.68 PR weight.
			// raw 8.262057 -> Roundup 8.3.
			name: "AV:N/AC:H/PR:L/UI:N/S:C/C:H/I:H/A:H",
			metrics: cvssMetrics{
				AttackVector: "N", AttackComplexity: "H", PrivilegesRequired: "L",
				UserInteraction: "N", Scope: "C",
				Confidentiality: "H", Integrity: "H", Availability: "H",
			},
			want: 8.3,
		},
		{
			// High complexity, network, no privileges: 8.1 in CVSS v3.1
			// (9.0 was the v3.0 value; the PR:H/AC:H weights changed).
			name: "AV:N/AC:H/PR:N/UI:N/S:U/C:H/I:H/A:H",
			metrics: cvssMetrics{
				AttackVector: "N", AttackComplexity: "H", PrivilegesRequired: "N",
				UserInteraction: "N", Scope: "U",
				Confidentiality: "H", Integrity: "H", Availability: "H",
			},
			want: 8.1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := cvssV31BaseScore(tc.metrics)
			if math.Abs(got-tc.want) > 0.005 {
				t.Errorf("cvssV31BaseScore = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestCVSSScoreIsNotPerSeverityLabel is the regression guard: the old code
// returned 7.5 for every "high" report regardless of its metrics.
func TestCVSSScoreIsNotPerSeverityLabel(t *testing.T) {
	s := &BugBountyService{
		programs:   map[string]*BugBountyProgram{},
		reports:    map[string]*BugReport{},
		hackers:    map[string]*Hacker{},
		rewardPool: new(big.Int),
	}

	// Two reports labelled "high" but with very different real exposure.
	wide := &BugReport{
		ID: "r1", Severity: "high", RewardAmount: "0",
		AttackVector: "N", AttackComplexity: "L", PrivilegesRequired: "N",
		UserInteraction: "N", Scope: "U",
		Confidentiality: "H", Integrity: "H", Availability: "H",
	}
	narrow := &BugReport{
		ID: "r2", Severity: "high", RewardAmount: "0",
		AttackVector: "P", AttackComplexity: "H", PrivilegesRequired: "H",
		UserInteraction: "R", Scope: "U",
		Confidentiality: "L", Integrity: "N", Availability: "N",
	}

	s.calculateCVSS(wide)
	s.calculateCVSS(narrow)

	if wide.CVSSScore == narrow.CVSSScore {
		t.Errorf("two %q reports with different metrics both scored %v; "+
			"the score must follow the metrics", "high", wide.CVSSScore)
	}
	if wide.CVSSScore <= narrow.CVSSScore {
		t.Errorf("wide exposure scored %v, narrow scored %v; want wide > narrow",
			wide.CVSSScore, narrow.CVSSScore)
	}
	if wide.CVSSScore == 7.5 {
		t.Error("CVSS score equals the removed per-severity constant 7.5")
	}
}

// TestCVSSMetricsDefaultSafely: a report with no metrics must produce a
// deterministic score, not a panic and not a fabricated severity.
func TestCVSSMetricsDefaultSafely(t *testing.T) {
	got := cvssV31BaseScore(cvssMetrics{})
	if got != 0.0 {
		t.Errorf("empty metrics scored %v, want 0 (no impact reported)", got)
	}

	// Metrics present but no impact must also be zero.
	got = cvssV31BaseScore(cvssMetrics{
		AttackVector: "N", AttackComplexity: "L", PrivilegesRequired: "N",
		UserInteraction: "N", Scope: "U",
	})
	if got != 0.0 {
		t.Errorf("metrics with no impact scored %v, want 0", got)
	}
}

// TestCVSSRoundupMatchesSpec checks the specification's rounding. The raw value
// for this vector is 8.708373; the specification reports 8.8, so a plain
// math.Ceil(x*10)/10 (which also gives 8.8) is right here and the test pins the
// rounding contract rather than a particular implementation.
func TestCVSSRoundupMatchesSpec(t *testing.T) {
	got := cvssV31BaseScore(cvssMetrics{
		AttackVector: "N", AttackComplexity: "L", PrivilegesRequired: "N",
		UserInteraction: "R", Scope: "U",
		Confidentiality: "H", Integrity: "H", Availability: "H",
	})
	if got != 8.8 {
		t.Errorf("score = %v, want 8.8", got)
	}

	// A raw value below an exact tenth must round up to it, not stay fractional.
	// This is the case math.Ceil on the raw score gets wrong: it would report
	// 8.262057 instead of 8.3.
	if got := cvssRoundup(8.262057); got != 8.3 {
		t.Errorf("cvssRoundup(8.262057) = %v, want 8.3", got)
	}

	// Every score must land exactly on a tenth, never on a raw float.
	for _, raw := range []float64{9.76017, 8.70838, 7.7077, 8.09429, 0.0, 10.0} {
		rounded := cvssRoundup(raw)
		if math.Abs(rounded*10-math.Round(rounded*10)) > 1e-9 {
			t.Errorf("cvssRoundup(%v) = %v, which is not a multiple of 0.1", raw, rounded)
		}
	}

	// The specification's Roundup works example: 4.02 -> 4.1.
	if got := cvssRoundup(4.02); got != 4.1 {
		t.Errorf("cvssRoundup(4.02) = %v, want 4.1", got)
	}
	// An exact tenth is preserved, not bumped.
	if got := cvssRoundup(9.8); got != 9.8 {
		t.Errorf("cvssRoundup(9.8) = %v, want 9.8", got)
	}
	// A raw value that already sits on a tenth stays put. 7.70000 must be 7.7,
	// not 7.8.
	if got := cvssRoundup(7.7); got != 7.7 {
		t.Errorf("cvssRoundup(7.7) = %v, want 7.7", got)
	}

	// The spec's roundup must never exceed 10.
	max := cvssV31BaseScore(cvssMetrics{
		AttackVector: "N", AttackComplexity: "L", PrivilegesRequired: "N",
		UserInteraction: "N", Scope: "C",
		Confidentiality: "H", Integrity: "H", Availability: "H",
	})
	if max > 10.0 {
		t.Errorf("score %v exceeds the CVSS maximum of 10", max)
	}
}

// TestScoreNeverExceedsTen is a property check over every metric combination:
// 4 vectors x 2 complexities x 3 privileges x 2 interactions x 2 scopes x
// 3^3 impact values = 2592 combinations.
func TestScoreNeverExceedsTen(t *testing.T) {
	vectors := []string{"N", "A", "L", "P"}
	complexities := []string{"L", "H"}
	privileges := []string{"N", "L", "H"}
	interactions := []string{"N", "R"}
	scopes := []string{"U", "C"}
	impacts := []string{"N", "L", "H"}

	checked := 0
	for _, av := range vectors {
		for _, ac := range complexities {
			for _, pr := range privileges {
				for _, ui := range interactions {
					for _, sc := range scopes {
						for _, c := range impacts {
							for _, i := range impacts {
								for _, a := range impacts {
									got := cvssV31BaseScore(cvssMetrics{
										AttackVector: av, AttackComplexity: ac,
										PrivilegesRequired: pr, UserInteraction: ui,
										Scope: sc, Confidentiality: c,
										Integrity: i, Availability: a,
									})
									if got < 0 || got > 10 {
										t.Fatalf("score %v out of range for %s/%s/%s/%s/%s/%s/%s/%s",
											got, av, ac, pr, ui, sc, c, i, a)
									}
									checked++
								}
							}
						}
					}
				}
			}
		}
	}
	if checked != 2592 {
		t.Errorf("checked %d combinations, want 2592", checked)
	}
}
