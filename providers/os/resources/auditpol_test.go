// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAuditpolInclusionAudits(t *testing.T) {
	cases := []struct {
		setting          string
		success, failure bool
	}{
		// English
		{"Success", true, false},
		{"Failure", false, true},
		{"Success and Failure", true, true},
		{"No Auditing", false, false},
		// German
		{"Erfolg", true, false},
		{"Fehler", false, true},
		{"Erfolg und Fehler", true, true},
		// Dutch
		{"Geslaagd", true, false},
		{"Mislukt", false, true},
		{"Geslaagd en mislukt", true, true},
		// Italian
		{"Operazione riuscita", true, false},
		{"Errore", false, true},
		{"Esito positivo e negativo", true, true},
		// French (with and without the accent on the capital É)
		{"Succès", true, false},
		{"Échec", false, true},
		{"Echec", false, true},
		{"Succès et échec", true, true},
		{"Succès et echec", true, true},
		// case-insensitive and whitespace-tolerant
		{"  success and failure  ", true, true},
		// unrecognized settings audit neither
		{"unbekannt", false, false},
		{"", false, false},
	}

	for _, c := range cases {
		t.Run(c.setting, func(t *testing.T) {
			flags := auditpolInclusionAudits(c.setting)
			assert.Equal(t, c.success, flags.success, "success")
			assert.Equal(t, c.failure, flags.failure, "failure")
		})
	}
}
