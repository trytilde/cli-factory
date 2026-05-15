package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cli-factory/internal/app"
	"cli-factory/internal/catalog"
	"cli-factory/internal/openai"
)

type toolItem struct {
	ID             string   `json:"id"`
	ProviderID     string   `json:"provider_id"`
	CommandPath    []string `json:"command_path"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Categories     []string `json:"categories"`
	SearchDocument string   `json:"search_document"`
}

type embeddingFile struct {
	Model     string          `json:"model"`
	Dimension int             `json:"dimension"`
	Items     []embeddingItem `json:"items"`
}

type embeddingItem struct {
	ID     string `json:"id"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

func main() {
	model := "text-embedding-3-small"
	dimensions := 768
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--model":
			i++
			if i < len(os.Args) {
				model = os.Args[i]
			}
		case "--dimensions":
			i++
			if i < len(os.Args) {
				parsed, err := strconv.Atoi(os.Args[i])
				if err != nil || parsed <= 0 {
					fail(fmt.Errorf("--dimensions must be a positive integer"))
				}
				dimensions = parsed
			}
		}
	}
	registry, err := app.Registry()
	if err != nil {
		fail(err)
	}
	var tools []toolItem
	for _, rt := range registry.Tools() {
		id := rt.Provider.ID() + "." + rt.Tool.ID()
		doc := strings.Join([]string{
			rt.Provider.ID(),
			rt.Provider.Name(),
			rt.Tool.ID(),
			rt.Tool.Name(),
			rt.Tool.ShortDescription(),
			rt.Tool.LongDescription(),
			strings.Join(rt.Tool.Categories(), " "),
			strings.Join(rt.Tool.Aliases(), " "),
		}, " ")
		tools = append(tools, toolItem{
			ID:             id,
			ProviderID:     rt.Provider.ID(),
			CommandPath:    []string{rt.Provider.ID(), rt.Tool.ID()},
			Name:           rt.Tool.Name(),
			Description:    rt.Tool.ShortDescription(),
			Categories:     rt.Tool.Categories(),
			SearchDocument: doc,
		})
	}
	if err := writeJSON(filepath.Join("catalog", "tools.json"), map[string]any{"items": tools}); err != nil {
		fail(err)
	}
	embedder := openai.Embedder{Model: model, Dimensions: dimensions}
	embeddings := embeddingFile{Model: model, Dimension: dimensions, Items: []embeddingItem{}}
	vectors := map[string][]float32{}
	if os.Getenv("OPENAI_API_KEY") != "" {
		for _, item := range tools {
			vec, err := embedder.Embed(context.Background(), item.SearchDocument)
			if err != nil {
				fail(err)
			}
			if len(vec) != dimensions {
				fail(fmt.Errorf("embedding for %s has dimension %d, want %d", item.ID, len(vec), dimensions))
			}
			converted := make([]float32, len(vec))
			for i, value := range vec {
				if math.IsNaN(value) || math.IsInf(value, 0) {
					fail(fmt.Errorf("embedding for %s contains invalid float", item.ID))
				}
				converted[i] = float32(value)
			}
			vectors[item.ID] = converted
			embeddings.Items = append(embeddings.Items, embeddingItem{
				ID:     item.ID,
				Offset: len(embeddings.Items) * dimensions,
				Length: dimensions,
			})
		}
	}
	if err := writeJSON(filepath.Join("catalog", "embeddings.json"), embeddings); err != nil {
		fail(err)
	}
	if err := writeEmbeddingBinary(filepath.Join("catalog", "embeddings.bin"), tools, vectors, dimensions); err != nil {
		fail(err)
	}
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func writeEmbeddingBinary(path string, tools []toolItem, vectors map[string][]float32, dimensions int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write([]byte(catalog.EmbeddingBinaryMagic)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(len(vectors))); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, uint32(dimensions)); err != nil {
		return err
	}
	for _, item := range tools {
		vec, ok := vectors[item.ID]
		if !ok {
			continue
		}
		for _, value := range vec {
			if err := binary.Write(file, binary.LittleEndian, value); err != nil {
				return err
			}
		}
	}
	return nil
}
