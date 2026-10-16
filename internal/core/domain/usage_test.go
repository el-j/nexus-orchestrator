package domain

import "testing"

func TestQuotaReadingRejected(t *testing.T) {
	if !(QuotaReading{Status: "rejected"}).Rejected() {
		t.Error("rejected")
	}
	for _, s := range []string{"allowed", "allowed_warning", ""} {
		if (QuotaReading{Status: s}).Rejected() {
			t.Errorf("%q is not rejected", s)
		}
	}
}
