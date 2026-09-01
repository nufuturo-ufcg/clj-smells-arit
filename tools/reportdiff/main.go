// Command reportdiff compares two Arit JSON reports without loading either
// report into memory in full. Reports are compared as ordered JSON arrays.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type comparison struct {
	Compared int
	Added    int
	Removed  int
	Changed  int
	Ordered  bool
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: reportdiff <previous.json> <current.json>")
		os.Exit(2)
	}

	previous, err := os.Open(os.Args[1])
	if err != nil {
		fail(err)
	}
	defer previous.Close()

	current, err := os.Open(os.Args[2])
	if err != nil {
		fail(err)
	}
	defer current.Close()

	result, err := compareReports(previous, current)
	if err != nil {
		fail(err)
	}

	fmt.Printf("compared=%d added=%d removed=%d changed=%d ordered=%t\n",
		result.Compared, result.Added, result.Removed, result.Changed, result.Ordered)
	if result.Added != 0 || result.Removed != 0 || result.Changed != 0 || !result.Ordered {
		os.Exit(1)
	}
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "reportdiff: %v\n", err)
	os.Exit(2)
}

func compareReports(previous, current io.Reader) (comparison, error) {
	previousDecoder := json.NewDecoder(previous)
	currentDecoder := json.NewDecoder(current)

	if err := expectArrayStart(previousDecoder); err != nil {
		return comparison{}, fmt.Errorf("previous report: %w", err)
	}
	if err := expectArrayStart(currentDecoder); err != nil {
		return comparison{}, fmt.Errorf("current report: %w", err)
	}

	result := comparison{Ordered: true}
	for {
		previousItem, previousDone, err := nextArrayItem(previousDecoder)
		if err != nil {
			return comparison{}, fmt.Errorf("previous report at index %d: %w", result.Compared, err)
		}
		currentItem, currentDone, err := nextArrayItem(currentDecoder)
		if err != nil {
			return comparison{}, fmt.Errorf("current report at index %d: %w", result.Compared, err)
		}

		if previousDone && currentDone {
			break
		}
		if previousDone {
			result.Added++
			remaining, err := drainArray(currentDecoder)
			if err != nil {
				return comparison{}, fmt.Errorf("current report: %w", err)
			}
			result.Added += remaining
			result.Ordered = false
			break
		}
		if currentDone {
			result.Removed++
			remaining, err := drainArray(previousDecoder)
			if err != nil {
				return comparison{}, fmt.Errorf("previous report: %w", err)
			}
			result.Removed += remaining
			result.Ordered = false
			break
		}

		result.Compared++
		if !bytes.Equal(previousItem, currentItem) {
			result.Changed++
			result.Ordered = false
		}
	}

	if err := expectEnd(previousDecoder); err != nil {
		return comparison{}, fmt.Errorf("previous report: %w", err)
	}
	if err := expectEnd(currentDecoder); err != nil {
		return comparison{}, fmt.Errorf("current report: %w", err)
	}
	return result, nil
}

func expectArrayStart(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '[' {
		return errors.New("expected a JSON array")
	}
	return nil
}

func nextArrayItem(decoder *json.Decoder) (json.RawMessage, bool, error) {
	if !decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return nil, false, err
		}
		if delimiter, ok := token.(json.Delim); !ok || delimiter != ']' {
			return nil, false, errors.New("expected end of JSON array")
		}
		return nil, true, nil
	}

	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return nil, false, err
	}
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, false, err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, false, err
	}
	return canonical, false, nil
}

func drainArray(decoder *json.Decoder) (int, error) {
	count := 0
	for decoder.More() {
		var ignored json.RawMessage
		if err := decoder.Decode(&ignored); err != nil {
			return count, err
		}
		count++
	}
	token, err := decoder.Token()
	if err != nil {
		return count, err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != ']' {
		return count, errors.New("expected end of JSON array")
	}
	return count, nil
}

func expectEnd(decoder *json.Decoder) error {
	var extra interface{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected trailing JSON value")
		}
		return err
	}
	return nil
}
