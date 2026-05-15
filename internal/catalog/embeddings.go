package catalog

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const EmbeddingBinaryMagic = "FCEMB001"

type EmbeddingManifest struct {
	Model     string                  `json:"model"`
	Dimension int                     `json:"dimension"`
	Items     []EmbeddingManifestItem `json:"items"`
}

type EmbeddingManifestItem struct {
	ID     string `json:"id"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

type EmbeddingIndex struct {
	Manifest EmbeddingManifest
	Vectors  map[string][]float32
}

func DecodeEmbeddingIndex(manifestData, binaryData []byte) (EmbeddingIndex, error) {
	var manifest EmbeddingManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return EmbeddingIndex{}, err
	}
	reader := bytes.NewReader(binaryData)
	magic := make([]byte, len(EmbeddingBinaryMagic))
	if _, err := io.ReadFull(reader, magic); err != nil {
		return EmbeddingIndex{}, err
	}
	if string(magic) != EmbeddingBinaryMagic {
		return EmbeddingIndex{}, fmt.Errorf("invalid embedding binary magic")
	}
	var count uint32
	var dimension uint32
	if err := binary.Read(reader, binary.LittleEndian, &count); err != nil {
		return EmbeddingIndex{}, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &dimension); err != nil {
		return EmbeddingIndex{}, err
	}
	if int(count) != len(manifest.Items) {
		return EmbeddingIndex{}, fmt.Errorf("embedding count mismatch: binary=%d manifest=%d", count, len(manifest.Items))
	}
	if int(dimension) != manifest.Dimension {
		return EmbeddingIndex{}, fmt.Errorf("embedding dimension mismatch: binary=%d manifest=%d", dimension, manifest.Dimension)
	}
	vectors := make(map[string][]float32, len(manifest.Items))
	for _, item := range manifest.Items {
		vec := make([]float32, dimension)
		for i := range vec {
			if err := binary.Read(reader, binary.LittleEndian, &vec[i]); err != nil {
				return EmbeddingIndex{}, err
			}
		}
		vectors[item.ID] = vec
	}
	return EmbeddingIndex{Manifest: manifest, Vectors: vectors}, nil
}
