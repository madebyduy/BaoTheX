package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"repwire/internal/briefmedia"
)

func main() {
	voice := flag.String("voice", "vi-VN-NamMinhNeural", "Edge voice name")
	output := flag.String("output", "var/edge-tts-smoke.mp3", "output MP3 path")
	text := flag.String("text", "Xin chào. Đây là bài kiểm tra giọng đọc Edge TTS bằng tiếng Việt.", "text to narrate")
	proxy := flag.String("proxy", "", "optional HTTP(S) proxy URL")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	duration, err := briefmedia.NewEdgeTTS(*voice, *proxy).Render(ctx, *text, *output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	info, err := os.Stat(*output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("ok output=%s bytes=%d duration_seconds=%d\n", *output, info.Size(), duration)
}
