package llx

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

const (
	scoreTypeMondoo = iota
	scoreTypeCVSSv3
)

func scoreVector(num int32) ([]byte, error) {
	if num > 100 || num < 0 {
		return nil, errors.New("Not a valid score (" + strconv.FormatInt(int64(num), 10) + ")")
	}

	return []byte{scoreTypeMondoo, byte(num & 0xff)}, nil
}

func scoreString(vector string) ([]byte, error) {
	switch {
	case strings.HasPrefix(vector, "CVSS:3.0/"):
		return cvssv3vector(vector[8:]), nil
	case strings.HasPrefix(vector, "CVSS:3.1/"):
		return cvssv3vector(vector[8:]), nil
	default:
		return nil, errors.New("Cannot parse this CVSS vector into a Mondoo score")
	}
}

// ScoreString turns a given score data into a printeable string
func ScoreString(b []byte) string {
	switch b[0] {
	case scoreTypeMondoo:
		s := strconv.Itoa(int(b[1]))
		return s
	case scoreTypeCVSSv3:
		num := cvssv3score(b)
		s := strconv.FormatFloat(num, 'f', 2, 64)
		return s + " (CVSSv3)"
	default:
		return "<unknown-score-type>"
	}
}

func scoreValue(vector []byte) (int, error) {
	switch vector[0] {
	case scoreTypeMondoo:
		return int(vector[1]), nil
	case scoreTypeCVSSv3:
		num := cvssv3score(vector)
		return int(100 - (num * 10)), nil
	default:
		return 0, errors.New("unknown score value")
	}
}

var (
	// all metrics are in order from highest to lowest; for example:
	// AV = [ Network, Adjacent, Local, Physical ]
	cvssv3multipliersAV     = []float64{0.85, 0.62, 0.55, 0.2}
	cvssv3multipliersAC     = []float64{0.77, 0.44}
	cvssv3multipliersPR     = []float64{0.85, 0.68, 0.5, 0.85, 0.62, 0.27} // 3 x changed, 3 x unchanged
	cvssv3multipliersUI     = []float64{0.85, 0.62}
	cvssv3multipliersCIA    = []float64{0.56, 0.22, 0}
	cvssv3multipliersE      = []float64{1, 1, 0.97, 0.94, 0.91}
	cvssv3multipliersRL     = []float64{1, 1, 0.97, 0.96, 0.95}
	cvssv3multipliersRC     = []float64{1, 1, 0.96, 0.92}
	cvssv3multipliersCRIRAR = []float64{1, 1.5, 1, 0.5}
)

func cvssv3vector(s string) []byte {
	res := make([]byte, 9)
	res[0] = scoreTypeCVSSv3

	// CVSS:3.1 /AV:_/AC:_/PR:_/UI:_/S:_/C:_/I:_/A:_
	//          0123456789_123456789_123456789_12345

	// AV:
	switch s[4] {
	case 'N', 'n':
		res[1] = 0
	case 'A', 'a':
		res[1] = 1
	case 'L', 'l':
		res[1] = 2
	case 'P', 'p':
		res[1] = 3
	}

	// AC:
	switch s[9] {
	case 'L', 'l':
		res[2] = 0
	case 'H', 'h':
		res[2] = 1
	}

	// PR:
	switch s[23] {
	case 'C':
		res[5] = 1
		switch s[14] {
		case 'N', 'n':
			res[3] = 0
		case 'L', 'l':
			res[3] = 1
		case 'H', 'h':
			res[3] = 2
		}
	case 'U':
		res[4] = 0
		switch s[14] {
		case 'N', 'n':
			res[3] = 3
		case 'L', 'l':
			res[3] = 4
		case 'H', 'h':
			res[3] = 5
		}
	}

	// UI:
	switch s[19] {
	case 'N', 'n':
		res[4] = 0
	case 'R', 'r':
		res[4] = 1
	}

	// C / I / A:
	for i := 0; i <= 2; i++ {
		switch s[27+i*4] {
		case 'H', 'h':
			res[6+i] = 0
		case 'L', 'l':
			res[6+i] = 1
		case 'N', 'n':
			res[6+i] = 2
		}
	}

	return res
}

// s: AV AC PR S UI C I A
func cvssv3score(s []byte) float64 {
	// Calculation:
	// https://www.first.org/cvss/specification-document
	// Chapter 7. CVSS v3.1 Equations

	c := cvssv3multipliersCIA[s[6]]
	i := cvssv3multipliersCIA[s[7]]
	a := cvssv3multipliersCIA[s[8]]
	iss := 1 - ((1 - c) * (1 - i) * (1 - a))

	var impact float64
	changed := s[5]
	if changed == 0 {
		impact = 6.42 * iss
	} else {
		impact = 7.52*(iss-0.029) - 3.25*math.Pow(iss-0.02, 15)
	}

	exploitability := 8.22 * cvssv3multipliersAV[s[1]] * cvssv3multipliersAC[s[2]] * cvssv3multipliersPR[s[3]] * cvssv3multipliersUI[s[4]]

	if iss <= 0 {
		return 0
	}

	var base float64
	if changed == 0 {
		base = impact + exploitability
	} else {
		base = 1.08 * (impact + exploitability)
	}

	return math.Ceil(math.Min(base, 10)*10.0) / 10.0
}
