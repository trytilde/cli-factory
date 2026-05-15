package catalog

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"
)

func TestDecodeEmbeddingIndex(t *testing.T) {
	manifest := EmbeddingManifest{
		Model:     "text-embedding-3-small",
		Dimension: 2,
		Items: []EmbeddingManifestItem{
			{ID: "example.echo", Offset: 0, Length: 2},
		},
	}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var bin bytes.Buffer
	bin.WriteString(EmbeddingBinaryMagic)
	if err := binary.Write(&bin, binary.LittleEndian, uint32(1)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(&bin, binary.LittleEndian, uint32(2)); err != nil {
		t.Fatal(err)
	}
	for _, value := range []float32{0.25, 0.75} {
		if err := binary.Write(&bin, binary.LittleEndian, value); err != nil {
			t.Fatal(err)
		}
	}
	index, err := DecodeEmbeddingIndex(manifestData, bin.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	got := index.Vectors["example.echo"]
	if len(got) != 2 || got[0] != 0.25 || got[1] != 0.75 {
		t.Fatalf("unexpected vector: %#v", got)
	}
}

func TestDecodeEmbeddingIndexRejectsMismatch(t *testing.T) {
	manifestData := []byte(`{"model":"text-embedding-3-small","dimension":3,"items":[]}`)
	var bin bytes.Buffer
	bin.WriteString(EmbeddingBinaryMagic)
	_ = binary.Write(&bin, binary.LittleEndian, uint32(0))
	_ = binary.Write(&bin, binary.LittleEndian, uint32(2))
	if _, err := DecodeEmbeddingIndex(manifestData, bin.Bytes()); err == nil {
		t.Fatal("expected mismatch error")
	}
}
