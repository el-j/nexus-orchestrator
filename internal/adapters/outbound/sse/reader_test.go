package sse

import (
	"errors"
	"strings"
	"testing"
)

func collect(t *testing.T, in string) []Event {
	t.Helper()
	var got []Event
	if err := Read(strings.NewReader(in), 1<<20, func(e Event) error {
		got = append(got, e)
		return nil
	}); err != nil {
		t.Fatalf("Read: %v", err)
	}
	return got
}

func TestReadSplitsEventsAndNames(t *testing.T) {
	got := collect(t, "event: a\ndata: 1\n\n: keep-alive\n\ndata:2\n\n")
	if len(got) != 2 || got[0] != (Event{Name: "a", Data: "1"}) || got[1] != (Event{Data: "2"}) {
		t.Fatalf("got %+v", got)
	}
}

func TestReadJoinsMultilineDataAndAcceptsCRLF(t *testing.T) {
	got := collect(t, "data: x\r\ndata: y\r\n\r\n")
	if len(got) != 1 || got[0].Data != "x\ny" {
		t.Fatalf("got %+v", got)
	}
}

func TestReadDeliversAnEventMissingItsBlankLine(t *testing.T) {
	got := collect(t, "data: last")
	if len(got) != 1 || got[0].Data != "last" {
		t.Fatalf("got %+v", got)
	}
}

func TestReadStopsOnStopAndPropagatesErrors(t *testing.T) {
	n := 0
	if err := Read(strings.NewReader("data: 1\n\ndata: 2\n\n"), 1<<20, func(Event) error { n++; return ErrStop }); err != nil || n != 1 {
		t.Fatalf("ErrStop: n=%d err=%v", n, err)
	}
	boom := errors.New("boom")
	if err := Read(strings.NewReader("data: 1\n\n"), 1<<20, func(Event) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
}

func TestReadFailsOnAnOversizedLine(t *testing.T) {
	err := Read(strings.NewReader("data: "+strings.Repeat("x", 100)+"\n\n"), 32, func(Event) error { return nil })
	if err == nil {
		t.Fatal("want a too-long-line error")
	}
}
