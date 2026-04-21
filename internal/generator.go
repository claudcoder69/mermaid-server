package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Generator provides the ability to generate a diagram.
type Generator interface {
	// Generate generates the given diagram.
	Generate(diagram *Diagram) error
	// CleanUp removes any diagrams that haven't been used within the given duration.
	CleanUp(duration time.Duration) error
}

// NewGenerator returns a generator that can be used to generate diagrams.
func NewGenerator(cache DiagramCache, mermaidCLIPath, inPath, outPath, puppeteerConfigPath string, logger *slog.Logger) Generator {
	return &cachingGenerator{
		cache:               cache,
		mermaidCLIPath:      mermaidCLIPath,
		inPath:              inPath,
		outPath:             outPath,
		puppeteerConfigPath: puppeteerConfigPath,
		logger:              logger,
	}
}

// cachingGenerator is an implementation of Generator.
type cachingGenerator struct {
	cache               DiagramCache
	mermaidCLIPath      string
	inPath              string
	outPath             string
	puppeteerConfigPath string
	logger              *slog.Logger
}

// Generate generates the given diagram.
func (c *cachingGenerator) Generate(diagram *Diagram) error {
	has, err := c.cache.Has(diagram)
	if err != nil {
		return fmt.Errorf("cache.Has failed: %w", err)
	}
	if has {
		cached, err := c.cache.Get(diagram)
		if err != nil {
			return fmt.Errorf("cache.Get failed: %w", err)
		}
		*diagram = *cached
		diagram.Touch()
		if err := c.cache.Store(diagram); err != nil {
			return fmt.Errorf("cache.Store failed: %w", err)
		}
		return nil
	}

	diagram.Touch()
	if err := c.generate(diagram); err != nil {
		return fmt.Errorf("generate failed: %w", err)
	}
	if err := c.cache.Store(diagram); err != nil {
		return fmt.Errorf("cache.Store failed: %w", err)
	}
	return nil
}

// generate does the actual file generation.
func (c *cachingGenerator) generate(diagram *Diagram) error {
	id, err := diagram.ID()
	if err != nil {
		return fmt.Errorf("cannot get diagram ID: %w", err)
	}

	inPath := filepath.Join(c.inPath, id+".mmd")
	outPath := filepath.Join(c.outPath, id+"."+diagram.imgType)

	if err := os.WriteFile(inPath, diagram.description, 0o644); err != nil {
		return fmt.Errorf("could not write to input file [%s]: %w", inPath, err)
	}

	if _, err := os.Stat(c.mermaidCLIPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("mermaid executable does not exist: %w", err)
		}
		return fmt.Errorf("could not stat mermaid executable: %w", err)
	}

	args := []string{"-i", inPath, "-o", outPath}
	if c.puppeteerConfigPath != "" {
		args = append(args, "-p", c.puppeteerConfigPath)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.mermaidCLIPath, args...)
	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed when executing mermaid: %w: stdout=%q stderr=%q", err, stdOut.String(), stdErr.String())
	}
	c.logger.Info("generated diagram", "id", id, "stdout", stdOut.String(), "stderr", stdErr.String())

	diagram.Output = outPath
	return nil
}

// CleanUp removes any diagrams that haven't been used within the given duration.
func (c *cachingGenerator) CleanUp(duration time.Duration) error {
	c.logger.Debug("running cleanup")
	diagrams, err := c.cache.GetAll()
	if err != nil {
		return fmt.Errorf("could not get cached diagrams: %w", err)
	}
	for _, d := range diagrams {
		if !d.TouchedInDuration(duration) {
			if err := c.delete(d); err != nil {
				return fmt.Errorf("could not delete diagram: %w", err)
			}
		}
	}
	return nil
}

// delete removes a single diagram and its files.
func (c *cachingGenerator) delete(diagram *Diagram) error {
	id, err := diagram.ID()
	if err != nil {
		return fmt.Errorf("cannot get diagram ID: %w", err)
	}

	c.logger.Info("cleaning up diagram", "id", id)

	inPath := filepath.Join(c.inPath, id+".mmd")
	outPath := filepath.Join(c.outPath, id+"."+diagram.imgType)

	if err := os.Remove(inPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("could not delete diagram input: %w", err)
	}
	if err := os.Remove(outPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("could not delete diagram output: %w", err)
	}
	if err := c.cache.Delete(diagram); err != nil {
		return fmt.Errorf("could not remove diagram from cache: %w", err)
	}
	return nil
}
