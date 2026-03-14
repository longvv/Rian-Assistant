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

	// Run espeak to generate WAV with smoother voice settings:
	//   -v en+f3   : use a softer female voice variant (less robotic than default)
	//   -s 140     : slower speaking rate (default 175) for more natural pacing
	//   -p 45      : slightly lower pitch for warmth
	//   -g 5       : small gap between words for natural breathing rhythm
	//   -w         : output to WAV file
	cmd := exec.CommandContext(ctx, "espeak",
		"-v", "en+f3",
		"-s", "140",
		"-p", "45",
		"-g", "5",
		"-w", wavPath,
		text,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("espeak failed: %w, output: %s", err, string(output))
	}

	logger.DebugCF("voice", "Converting WAV to OGG Opus via ffmpeg with audio smoothing", map[string]interface{}{})

	// Run ffmpeg to convert WAV to OGG Opus with audio post-processing:
	//   lowpass=4000  : cut harsh high-frequency artifacts typical of espeak
	//   aresample=48000 : resample to 48kHz for Opus codec quality
	//   -b:a 64k     : higher bitrate for better audio fidelity
	cmd = exec.CommandContext(ctx, "ffmpeg", "-y",
		"-i", wavPath,
		"-af", "lowpass=f=4000,aresample=48000",
		"-c:a", "libopus",
		"-b:a", "64k",
		oggPath,
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("ffmpeg failed: %w, output: %s", err, string(output))
	}

	return oggPath, nil
}
