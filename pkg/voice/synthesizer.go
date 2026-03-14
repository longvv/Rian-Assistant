package voice

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/logger"
)

type Synthesizer interface {
	Synthesize(ctx context.Context, text string) (string, error)
	IsAvailable() bool
}

type EspeakSynthesizer struct {
	available bool
}

func NewEspeakSynthesizer() *EspeakSynthesizer {
	// Check if espeak and ffmpeg are installed
	err := exec.Command("espeak", "--version").Run()
	hasEspeak := err == nil

	err = exec.Command("ffmpeg", "-version").Run()
	hasFfmpeg := err == nil

	available := hasEspeak && hasFfmpeg
	if !available {
		logger.WarnCF("voice", "EspeakSynthesizer dependencies missing", map[string]interface{}{
			"espeak": hasEspeak,
			"ffmpeg": hasFfmpeg,
		})
	}

	return &EspeakSynthesizer{
		available: available,
	}
}

func (s *EspeakSynthesizer) IsAvailable() bool {
	return s.available
}

func (s *EspeakSynthesizer) Synthesize(ctx context.Context, text string) (string, error) {
	if !s.available {
		return "", fmt.Errorf("espeak or ffmpeg not available")
	}

	// Create a temporary directory for the audio files
	tmpDir, err := os.MkdirTemp("", "tts-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	wavPath := filepath.Join(tmpDir, "out.wav")
	oggPath := filepath.Join(tmpDir, "out.ogg")

	logger.DebugCF("voice", "Starting espeak synthesis", map[string]interface{}{
		"text_len": len(text),
	})

	// Run espeak to generate WAV
	// -w specifies the output file
	cmd := exec.CommandContext(ctx, "espeak", "-w", wavPath, text)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("espeak failed: %w, output: %s", err, string(output))
	}

	logger.DebugCF("voice", "Converting WAV to OGG Opus via ffmpeg", map[string]interface{}{})

	// Run ffmpeg to convert WAV to OGG Opus
	cmd = exec.CommandContext(ctx, "ffmpeg", "-y", "-i", wavPath, "-c:a", "libopus", "-b:a", "32k", oggPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("ffmpeg failed: %w, output: %s", err, string(output))
	}

	return oggPath, nil
}
