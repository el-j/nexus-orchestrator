package mcp

import "testing"

func TestMCPErrorCarriesItsMessage(t *testing.T) {
	var err error = &mcpError{code: codeInvalidParams, msg: "bad things"}
	if err.Error() != "bad things" {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestCaptureWriterRecordsStatusAndBody(t *testing.T) {
	c := &captureWriter{}
	c.WriteHeader(202)
	if _, err := c.Write([]byte("ab")); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Write([]byte("cd")); err != nil {
		t.Fatal(err)
	}
	if c.status != 202 || string(c.body) != "abcd" || c.Header() == nil {
		t.Errorf("%+v", c)
	}
}
