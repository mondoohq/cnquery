package cvss

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative cvss.proto
//go:generate go run golang.org/x/tools/cmd/stringer -type=Severity

var Metrics map[string][]string

// 5.8/AV:N/AC:M/Au:N/C:P/I:P/A:N
func init() {
	// defines the valid metrics
	// CVSS v3 https://www.first.org/cvss/specification-document#2-Base-Metrics
	// CVSS v2 https://www.first.org/cvss/v2/guide
	Metrics = map[string][]string{
		// Version
		"CVSS": {"3.0", "3.1"},
		// Attack Vector
		"AV": {"N", "A", "L", "P"},
		// Attack Complexity
		"AC": {
			// CVSS 3.0
			"L", "H",
			// CVSS 2.0
			"M",
		},
		// Privileges Required
		"PR": {"N", "L", "H"},
		// User Interaction
		"UI": {"N", "R"},
		// Scope
		"S": {"U", "C"},
		// Confidentiality Impact
		"C": {
			// CVSS 3.0
			"H", "L", "N",
			// CVSS 2.0
			"P", "C",
		},
		//  Integrity Impact
		"I": {
			// CVSS 3.0
			"H", "L", "N",
			// CVSS 2.0
			"P", "C",
		},
		// Availability Impact
		"A": {
			// CVSS 3.0
			"H", "L", "N",
			// CVSS 2.0
			"P", "C",
		},
		// Exploit Code Maturity
		"E": {
			// CVSS 3.0
			"X", "H", "F", "P", "U",
			// CVSS 2.0
			"POC", "ND",
		},
		// Remediation Level
		"RL": {
			// CVSS 3.0
			"X", "U", "W", "T", "O",
			// CVSS 2.0
			"OF", "TF", "ND",
		},
		// Report Confidence
		"RC": {
			// CVSS 3.0
			"X", "C", "R", "U",
			// CVSS 2.0
			"UC", "UR", "ND",
		},
		// Confidentiality Requirement
		"CR": {
			// CVSS 3.0
			"X", "H", "M", "L",
			// CVSS 2.0
			"ND",
		},
		// Integrity Req
		"IR": {
			// CVSS 3.0
			"X", "H", "M", "L",
			// CVSS 2.0
			"ND",
		},
		// Availability Req
		"AR": {
			// CVSS 3.0
			"X", "H", "M", "L",
			// CVSS 2.0
			"ND",
		},

		// Authentication, CVSS 2.0 only
		// https://www.first.org/cvss/v2/guide#2-1-3-Authentication-Au
		"AU": {
			"M", "S", "N",
		},
		// https://www.first.org/cvss/v2/guide#2-3-1-Collateral-Damage-Potential-CDP
		"CDP": {
			"N", "L", "LM", "MH", "H", "ND",
		},
		// https://www.first.org/cvss/v2/guide#2-3-2-Target-Distribution-TD
		"TD": {
			"M", "L", "M", "H", "ND",
		},
	}
}

const NoneVector = "0.0/CVSS:3.0"

var CVSS_VERSION = regexp.MustCompile(`^.*\/CVSS:([\d.]+)(?:\/.*)*$`)

func New(vector string) (*Cvss, error) {
	if len(vector) == 0 {
		return nil, errors.New("vector cannot be empty")
	}

	// trim whitespace
	vector = strings.TrimSpace(vector)

	c := &Cvss{Vector: vector}

	// ensure score field is set
	c.Score = c.DetermineScore()

	// check that the vector is parsable and the metrics are correct
	if !c.Verify() {
		return nil, errors.New("cvss vector is not parsable or valid: " + vector)
	}

	return c, nil
}

func (c *Cvss) Version() string {
	m := CVSS_VERSION.FindStringSubmatch(c.Vector)

	if len(m) == 2 {
		return m[1]
	} else {
		return "2.0"
	}
}

func (c *Cvss) DetermineScore() float32 {
	var err error
	vector := c.Vector
	pairs := strings.Split(vector, "/")

	if len(pairs) < 1 {
		c.Score = float32(0.0)
	}

	// first entry includes the score
	var score float64
	if score, err = strconv.ParseFloat(pairs[0], 32); err != nil {
		// error handling, fallback to default value
		return float32(0.0)
	}

	c.Score = float32(score)
	return c.Score
}

func (c *Cvss) Metrics() (map[string]string, error) {
	values := make(map[string]string)

	vector := c.Vector
	pairs := strings.Split(vector, "/")

	if len(pairs) < 1 {
		return nil, errors.New("invalid cvss string: " + vector)
	}

	// parse the key values
	for i, entry := range pairs {
		// ignore first entry which is a score, do not save it here to avoid side-effects
		// functionality has moved ParseScore
		if i == 0 {
			continue
		}

		// likely an entry  with trailing slash (6.5/AV:N/AC:L/Au:S/C:P/I:P/A:P/), that is okay
		if len(entry) == 0 {
			continue
		}

		// split key value
		kv := strings.Split(entry, ":")
		if len(kv) < 2 {
			log.Debug().Str("vector", vector).Msg("could not parse vector properly")
		} else {
			values[strings.ToUpper(kv[0])] = strings.ToUpper(kv[1])
		}
	}

	return values, nil
}

// Severity converts the CVSS Score (0.0 - 10.0) as specified in CVSS v3.0
// specification (https://www.first.org/cvss/specification-document) table 14
// to qualitative severity rating scale
func (c *Cvss) Severity() Severity {
	return Rating(c.Score)
}

func (c *Cvss) Verify() bool {
	values, err := c.Metrics()
	if err != nil {
		return false
	}

	for k, v := range values {
		values, ok := Metrics[k]
		if !ok {
			return false
		}
		if !contains(values, v) {
			return false
		}
	}
	return true
}

func contains(slice []string, search string) bool {
	for _, value := range slice {
		if value == search {
			return true
		}
	}
	return false
}

// Compare returns an integer comparing two cvss scores
// The result will be 0 if a==b, -1 if a < b, and +1 if a > b
func (c *Cvss) Compare(d *Cvss) int {
	if c.Score == d.Score {
		return 0
	} else if c.Score < d.Score {
		return -1
	} else {
		return 1
	}
}

func Rating(score float32) Severity {
	switch {
	case score == 0.0:
		return None
	case score > 0 && score < 4.0:
		return Low
	case score >= 4.0 && score < 7.0:
		return Medium
	case score >= 7.0 && score < 9.0:
		return High
	case score >= 9.0:
		return Critical
	}
	// negative numbers may be used for no-parsable cvss vectors
	return Unknown
}

// Severity defines the cvss v3 range
// in addition is defines an additional state unknown
// iota is sorted by criticality to ease easy int comparison to detect the severity with the
// highest criticality
type Severity int

const (
	Unknown  Severity = iota // could not be determined
	None                     // 0.0, e.g. mapped ubuntu negligible is mapped to none
	Low                      // 0.1 - 3.9
	Medium                   // 4.0 - 6.9
	High                     // 7.0 - 8.9
	Critical                 // 9.0 - 10.0
)

func MaxScore(cvsslist []*Cvss) (*Cvss, error) {
	none, _ := New(NoneVector)

	// no entry, no return :-)
	if len(cvsslist) == 0 {
		return none, nil
	}

	res := cvsslist[0]

	// easy, we just have one entry
	if len(cvsslist) == 1 {
		return res, nil
	}

	// fun starts, we need to compare cvss scores now
	max := res
	maxScore, err := New(max.Vector)
	if err != nil {
		return none, err
	}

	for i := 1; i < len(cvsslist); i++ {
		entry := cvsslist[i]
		vector := entry.Vector
		score, err := New(vector)
		if err != nil {
			return none, err
		}

		if maxScore.Compare(score) < 0 {
			max = entry
			maxScore = score
		}
	}

	return max, nil
}
