package ffmpeg

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const (
	framesPerSecond    = "1"
	framePattern       = "frame_%04d.png"
	frameGlob          = "*.png"
	outputPrefix       = "outputs"
	outputContentType  = "application/zip"
	directoryMode      = 0o755
	inputFileBaseName  = "input"
	framesSubdirectory = "frames"
	zipFileName        = "frames.zip"
)

type ObjectStore interface {
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
}

type Extractor struct {
	store ObjectStore
}

func NewExtractor(store ObjectStore) *Extractor {
	return &Extractor{store: store}
}

func (e *Extractor) ExtractFrames(ctx context.Context, source string) (string, int64, int, error) {
	workDir, err := os.MkdirTemp("", "extract-")
	if err != nil {
		return "", 0, 0, err
	}
	defer os.RemoveAll(workDir)

	inputPath := filepath.Join(workDir, inputFileBaseName+path.Ext(source))
	if err := e.download(ctx, source, inputPath); err != nil {
		return "", 0, 0, err
	}

	framesDir := filepath.Join(workDir, framesSubdirectory)
	if err := os.MkdirAll(framesDir, directoryMode); err != nil {
		return "", 0, 0, err
	}
	if err := runFFmpeg(ctx, inputPath, filepath.Join(framesDir, framePattern)); err != nil {
		return "", 0, 0, err
	}

	frames, err := filepath.Glob(filepath.Join(framesDir, frameGlob))
	if err != nil {
		return "", 0, 0, err
	}
	if len(frames) == 0 {
		return "", 0, 0, fmt.Errorf("nenhum frame extraído do vídeo")
	}

	zipPath := filepath.Join(workDir, zipFileName)
	if err := writeZip(frames, zipPath); err != nil {
		return "", 0, 0, err
	}

	zipKey := outputKeyFor(source)
	size, err := e.upload(ctx, zipKey, zipPath)
	if err != nil {
		return "", 0, 0, err
	}
	return zipKey, size, len(frames), nil
}

func outputKeyFor(source string) string {
	base := strings.TrimSuffix(path.Base(source), path.Ext(source))
	return path.Join(outputPrefix, base+".zip")
}

func (e *Extractor) download(ctx context.Context, key, dest string) error {
	reader, err := e.store.Get(ctx, key)
	if err != nil {
		return err
	}
	defer reader.Close()

	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

func (e *Extractor) upload(ctx context.Context, key, sourcePath string) (int64, error) {
	file, err := os.Open(sourcePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	if err := e.store.Put(ctx, key, file, info.Size(), outputContentType); err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func runFFmpeg(ctx context.Context, input, outputPattern string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", input, "-vf", "fps="+framesPerSecond, "-y", outputPattern)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg falhou: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func writeZip(files []string, dest string) error {
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	writer := zip.NewWriter(out)
	defer writer.Close()

	for _, file := range files {
		if err := addToZip(writer, file); err != nil {
			return err
		}
	}
	return nil
}

func addToZip(writer *zip.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	entry, err := writer.Create(filepath.Base(filename))
	if err != nil {
		return err
	}
	_, err = io.Copy(entry, file)
	return err
}
