package pipeline

import (
	"context"
	"errors"
	"testing"
)

func TestRunEmpty(t *testing.T) {
	p := New()
	r := &Result{}
	if err := p.Run(context.Background(), r); err != nil {
		t.Fatalf("empty pipeline returned error: %v", err)
	}
}

func TestStagesRunInOrder(t *testing.T) {
	var order []int

	s1 := func(_ context.Context, _ *Result) error { order = append(order, 1); return nil }
	s2 := func(_ context.Context, _ *Result) error { order = append(order, 2); return nil }
	s3 := func(_ context.Context, _ *Result) error { order = append(order, 3); return nil }

	p := New(s1, s2, s3)
	if err := p.Run(context.Background(), &Result{}); err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("stages ran out of order: %v", order)
	}
}

func TestErrorStopsPipeline(t *testing.T) {
	boom := errors.New("boom")
	ran := false

	s1 := func(_ context.Context, _ *Result) error { return boom }
	s2 := func(_ context.Context, _ *Result) error { ran = true; return nil }

	p := New(s1, s2)
	err := p.Run(context.Background(), &Result{})
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom error, got: %v", err)
	}
	if ran {
		t.Fatal("second stage should not have run after error")
	}
}

func TestCancelledStopsPipeline(t *testing.T) {
	ran := false

	s1 := func(_ context.Context, r *Result) error { r.Cancelled = true; return nil }
	s2 := func(_ context.Context, _ *Result) error { ran = true; return nil }

	p := New(s1, s2)
	r := &Result{}
	if err := p.Run(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if ran {
		t.Fatal("second stage should not have run after cancellation")
	}
	if !r.Cancelled {
		t.Fatal("result should be marked cancelled")
	}
}

func TestStagesModifyResult(t *testing.T) {
	s1 := func(_ context.Context, r *Result) error {
		r.RawText = "hello world"
		return nil
	}
	s2 := func(_ context.Context, r *Result) error {
		r.OutputText = r.RawText + "!"
		return nil
	}

	r := &Result{}
	p := New(s1, s2)
	if err := p.Run(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if r.OutputText != "hello world!" {
		t.Fatalf("unexpected output: %q", r.OutputText)
	}
}

func TestMetadataPassesBetweenStages(t *testing.T) {
	s1 := func(_ context.Context, r *Result) error {
		r.Metadata = map[string]string{"key": "value"}
		return nil
	}
	s2 := func(_ context.Context, r *Result) error {
		if r.Metadata["key"] != "value" {
			return errors.New("metadata not visible")
		}
		return nil
	}

	if err := New(s1, s2).Run(context.Background(), &Result{}); err != nil {
		t.Fatal(err)
	}
}

func TestModeConstants(t *testing.T) {
	if ModeDictation != 0 || ModeCommand != 1 || ModePrompt != 2 {
		t.Fatal("mode constants have unexpected values")
	}
}

func TestContextCancellationStopsPipeline(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	ran := false
	s := func(_ context.Context, _ *Result) error { ran = true; return nil }

	p := New(s)
	err := p.Run(ctx, &Result{})
	if err == nil {
		t.Fatal("expected context.Canceled error, got nil")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got: %v", err)
	}
	if ran {
		t.Fatal("stage should not have run after context cancellation")
	}
}

func TestNilStagePanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil stage, got none")
		}
	}()
	New(nil)
}
