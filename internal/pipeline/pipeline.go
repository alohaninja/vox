package pipeline

import (
	"context"
	"fmt"
)

// Mode represents the type of voice input.
type Mode int

const (
	// ModeDictation is standard speech-to-text dictation.
	ModeDictation Mode = iota
	// ModeCommand triggers a voice command (e.g., "create PR", "query flag").
	ModeCommand
	// ModePrompt triggers a Claude prompt shortcut (e.g., "summarize my clipboard").
	ModePrompt
)

// Result flows through the pipeline. Each stage can read and modify it.
type Result struct {
	RawAudio      []byte
	RawText       string
	ProcessedText string
	OutputText    string
	Mode          Mode
	Cancelled     bool
	Metadata      map[string]string
}

// Stage is a function that processes a Result. Stages are chained in a Pipeline.
// A stage can modify the Result, set Cancelled to true to stop further processing,
// or return an error.
type Stage func(ctx context.Context, r *Result) error

// Pipeline runs a sequence of stages on a Result.
type Pipeline struct {
	stages []Stage
}

// New creates a Pipeline with the given stages. It panics if any stage is nil.
func New(stages ...Stage) *Pipeline {
	for i, s := range stages {
		if s == nil {
			panic(fmt.Sprintf("pipeline: stage %d is nil", i))
		}
	}
	return &Pipeline{stages: stages}
}

// Run executes each stage in order. It stops early if the context is
// cancelled, a stage returns an error, or a stage sets r.Cancelled to true.
func (p *Pipeline) Run(ctx context.Context, r *Result) error {
	for _, stage := range p.stages {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := stage(ctx, r); err != nil {
			return err
		}
		if r.Cancelled {
			return nil
		}
	}
	return nil
}
