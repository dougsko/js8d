package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/js8call/js8d/pkg/dsp"
)

func main() {
	var (
		message    = flag.String("message", "", "JS8 message to encode (max 12 chars)")
		sampleRate = flag.Int("rate", 12000, "Audio sample rate")
		fillChar   = flag.String("fill", "-", "Character to pad short messages")
		frameType  = flag.Int("type", 0, "JS8 frame type (0-7)")
		output     = flag.String("output", "", "Output audio file (raw 16-bit samples)")
		showTones  = flag.Bool("tones", false, "Show tone sequence")
	)
	flag.Parse()

	if *message == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -message \"CQ N0CALL\" [options]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Validate and pad message
	if len(*fillChar) != 1 {
		fmt.Fprintf(os.Stderr, "Fill character must be exactly 1 character\n")
		os.Exit(1)
	}

	if err := dsp.ValidateMessage(*fillChar); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid fill character: %v\n", err)
		os.Exit(1)
	}

	// Pad message to 12 characters
	paddedMsg, err := dsp.PadMessage(*message, (*fillChar)[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Message error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Encoding JS8 Message\n")
	fmt.Printf("====================\n")
	fmt.Printf("Original: %q\n", *message)
	fmt.Printf("Padded:   %q\n", paddedMsg)
	fmt.Printf("Length:   %d characters\n", len(paddedMsg))
	fmt.Printf("Type:     %d\n", *frameType)
	fmt.Printf("Rate:     %d Hz\n", *sampleRate)
	fmt.Printf("\n")

	// Create encoder
	encoder := dsp.NewJS8Encoder()

	// Encode to tones
	tones, err := encoder.EncodeMessage(paddedMsg, *frameType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Encoding failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Encoded to %d tones\n", len(tones))

	if *showTones {
		fmt.Printf("\nTone Sequence:\n")
		fmt.Printf("=============\n")
		for i, tone := range tones {
			marker := ""
			if i < 7 || (i >= 36 && i < 43) || i >= 72 {
				marker = " (Costas)"
			} else if i >= 7 && i < 36 {
				marker = " (Parity)"
			} else {
				marker = " (Data)"
			}
			fmt.Printf("%2d: %d%s\n", i, tone, marker)
		}
		fmt.Printf("\n")
	}

	// Generate audio
	audio := encoder.GenerateAudio(tones, *sampleRate)
	duration := float64(len(audio)) / float64(*sampleRate)

	fmt.Printf("✓ Generated %d audio samples (%.2f seconds)\n", len(audio), duration)

	// Calculate some statistics
	var minSample, maxSample int16 = 32767, -32768
	var avgSample float64
	for _, sample := range audio {
		if sample < minSample {
			minSample = sample
		}
		if sample > maxSample {
			maxSample = sample
		}
		avgSample += float64(sample)
	}
	avgSample /= float64(len(audio))

	fmt.Printf("Audio Stats:\n")
	fmt.Printf("  Range:    %d to %d\n", minSample, maxSample)
	fmt.Printf("  Average:  %.1f\n", avgSample)
	fmt.Printf("  Peak:     %.1f%% of full scale\n", float64(maxSample)/32767.0*100)

	// Output to file if requested
	if *output != "" {
		file, err := os.Create(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create output file: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		// Write raw 16-bit samples (little endian)
		for _, sample := range audio {
			file.Write([]byte{byte(sample), byte(sample >> 8)})
		}

		fmt.Printf("✓ Wrote audio to %s\n", *output)
		fmt.Printf("  Play with: sox -r %d -e signed -b 16 -c 1 %s -t alsa\n", *sampleRate, *output)
	}

	fmt.Printf("\nJS8 Message Breakdown:\n")
	fmt.Printf("=====================\n")
	fmt.Printf("Total transmission: %.1f seconds\n", duration)
	fmt.Printf("79 tones = 7 + 29 + 7 + 29 + 7\n")
	fmt.Printf("  Start Costas:  7 tones (sync)\n")
	fmt.Printf("  Parity data:  29 tones (error correction)\n")
	fmt.Printf("  Middle Costas: 7 tones (sync)\n")
	fmt.Printf("  Message data: 29 tones (your message)\n")
	fmt.Printf("  End Costas:    7 tones (sync)\n")
	fmt.Printf("\nFrequency plan:\n")
	fmt.Printf("  Base freq:    1000 Hz\n")
	fmt.Printf("  Tone spacing: %.2f Hz\n", float64(*sampleRate)/2048.0)
	fmt.Printf("  Bandwidth:    ~%.0f Hz (8 tones)\n", 8*float64(*sampleRate)/2048.0)

	// Show first few frequency values
	fmt.Printf("\nFirst few tone frequencies:\n")
	baseFreq := 1000.0
	freqSpacing := float64(*sampleRate) / 2048.0
	for i := 0; i < 8 && i < len(tones); i++ {
		freq := baseFreq + float64(tones[i])*freqSpacing
		fmt.Printf("  Tone %d: %.2f Hz\n", tones[i], freq)
	}
}