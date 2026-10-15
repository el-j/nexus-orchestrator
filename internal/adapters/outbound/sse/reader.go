// Package sse reads server-sent event streams (text/event-stream) for the LLM adapters.
package sse

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// Event is one dispatched server-sent event.
type Event struct {
	// Name is the "event:" field, empty when the server sent none.
	Name string
	// Data is the "data:" lines joined with newlines.
	Data string
}

// ErrStop is returned by a Read callback to end the stream early without an error.
var ErrStop = errors.New("sse: stop")

// Read parses r and calls fn for every event. Lines longer than maxLine bytes are an error.
// An event whose terminating blank line is missing at the end of the stream is still delivered.
// Read returns nil when the stream ends or fn returns ErrStop, and fn's error otherwise.
func Read(r io.Reader, maxLine int, fn func(Event) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, min(64<<10, maxLine)), maxLine)
	var (
		name string
		data []string
	)
	dispatch := func() error {
		if len(data) == 0 {
			name = ""
			return nil
		}
		ev := Event{Name: name, Data: strings.Join(data, "\n")}
		name, data = "", nil
		err := fn(ev)
		if errors.Is(err, ErrStop) {
			return ErrStop
		}
		return err
	}
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		switch {
		case line == "":
			if err := dispatch(); err != nil {
				return stopToNil(err)
			}
		case strings.HasPrefix(line, ":"): // comment / keep-alive
		case strings.HasPrefix(line, "event:"):
			name = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return stopToNil(dispatch())
}

func stopToNil(err error) error {
	if errors.Is(err, ErrStop) {
		return nil
	}
	return err
}
