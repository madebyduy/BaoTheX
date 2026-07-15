package briefmedia

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type VideoStory struct {
	Title  string
	Source string
}

type VideoRenderer struct {
	FFmpeg   string
	FontFile string
}

func NewVideoRenderer(ffmpeg, fontFile string) *VideoRenderer {
	return &VideoRenderer{FFmpeg: ffmpeg, FontFile: fontFile}
}

func (v *VideoRenderer) Enabled() bool {
	return strings.TrimSpace(v.FFmpeg) != "" && strings.TrimSpace(v.FontFile) != ""
}

// Render produces a 1280x720 MP4 from headline slides and a narrated WAV.
func (v *VideoRenderer) Render(ctx context.Context, title string, stories []VideoStory, audioPath, outputPath, thumbnailPath string, audioDuration int) error {
	if !v.Enabled() || len(stories) == 0 {
		return fmt.Errorf("video renderer is not configured")
	}
	if _, err := os.Stat(v.FontFile); err != nil {
		return fmt.Errorf("video font: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	workDir := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + "-render"
	if err := os.RemoveAll(workDir); err != nil {
		return err
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	seconds := int(math.Ceil(float64(max(audioDuration, 35)) / float64(len(stories))))
	segments := make([]string, 0, len(stories))
	for i, story := range stories {
		titleFile := filepath.Join(workDir, fmt.Sprintf("title-%02d.txt", i))
		metaFile := filepath.Join(workDir, fmt.Sprintf("meta-%02d.txt", i))
		if err := os.WriteFile(titleFile, []byte(wrapHeadline(story.Title, 34)), 0o644); err != nil {
			return err
		}
		meta := strings.ToUpper(story.Source) + "  ·  " + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(stories))
		if err := os.WriteFile(metaFile, []byte(meta), 0o644); err != nil {
			return err
		}
		segment := filepath.Join(workDir, fmt.Sprintf("segment-%02d.mp4", i))
		filter := strings.Join([]string{
			"drawbox=x=0:y=0:w=iw:h=12:color=0xff684d:t=fill",
			"drawbox=x=72:y=145:w=90:h=5:color=0xff684d:t=fill",
			drawText(v.FontFile, metaFile, 26, "0xff8d77", 80, 85),
			drawText(v.FontFile, titleFile, 58, "white", 80, 185),
			"drawtext=fontfile='" + filterPath(v.FontFile) + "':text='BaoTheX':fontcolor=white:fontsize=30:x=80:y=h-90",
			"drawtext=fontfile='" + filterPath(v.FontFile) + "':text='TIN NHANH TRONG NGAY':fontcolor=0x9ba7b8:fontsize=20:x=w-tw-80:y=h-84",
		}, ",")
		args := []string{"-y", "-f", "lavfi", "-i", "color=c=0x090c12:s=1280x720:r=30:d=" + strconv.Itoa(seconds),
			"-vf", filter, "-c:v", "libx264", "-preset", "veryfast", "-crf", "22", "-pix_fmt", "yuv420p", segment}
		if err := runFFmpeg(ctx, v.FFmpeg, args...); err != nil {
			return err
		}
		segments = append(segments, segment)
	}

	concatFile := filepath.Join(workDir, "segments.txt")
	var concat strings.Builder
	for _, segment := range segments {
		absoluteSegment, err := filepath.Abs(segment)
		if err != nil {
			return err
		}
		concat.WriteString("file '")
		concat.WriteString(strings.ReplaceAll(filepath.ToSlash(absoluteSegment), "'", "'\\''"))
		concat.WriteString("'\n")
	}
	if err := os.WriteFile(concatFile, []byte(concat.String()), 0o644); err != nil {
		return err
	}
	if err := runFFmpeg(ctx, v.FFmpeg, "-y", "-f", "concat", "-safe", "0", "-i", concatFile,
		"-i", audioPath, "-c:v", "copy", "-c:a", "aac", "-b:a", "160k", "-shortest", "-movflags", "+faststart", outputPath); err != nil {
		return err
	}
	if err := runFFmpeg(ctx, v.FFmpeg, "-y", "-ss", "0.2", "-i", outputPath, "-frames:v", "1", "-q:v", "2", thumbnailPath); err != nil {
		return err
	}
	_ = title
	return nil
}

func drawText(font, textFile string, size int, color string, x, y int) string {
	return "drawtext=fontfile='" + filterPath(font) + "':textfile='" + filterPath(textFile) +
		"':fontcolor=" + color + ":fontsize=" + strconv.Itoa(size) + ":line_spacing=16:x=" + strconv.Itoa(x) + ":y=" + strconv.Itoa(y)
}

func filterPath(path string) string {
	path = filepath.ToSlash(path)
	path = strings.ReplaceAll(path, "'", "\\'")
	return strings.ReplaceAll(path, ":", "\\:")
}

func wrapHeadline(text string, width int) string {
	words := strings.Fields(text)
	var lines []string
	var line string
	for _, word := range words {
		if line != "" && len([]rune(line))+1+len([]rune(word)) > width {
			lines = append(lines, line)
			line = word
		} else if line == "" {
			line = word
		} else {
			line += " " + word
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) > 5 {
		lines = lines[:5]
		lines[4] = strings.TrimSuffix(lines[4], ".") + "…"
	}
	return strings.Join(lines, "\n")
}

func runFFmpeg(ctx context.Context, binary string, args ...string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := string(out)
		if len(message) > 1800 {
			message = message[len(message)-1800:]
		}
		return fmt.Errorf("ffmpeg: %w: %s", err, message)
	}
	return nil
}
