package repo_sqlite

import (
	"math"
	"testing"
)

func TestInternal_EncodeDecodeEmbedding(t *testing.T) {
	// Nil / empty slice
	if b := encodeEmbedding(nil); b != nil {
		t.Errorf("expected nil byte slice, got %v", b)
	}
	if b := encodeEmbedding([]float32{}); b != nil {
		t.Errorf("expected nil byte slice, got %v", b)
	}
	if v := decodeEmbedding(nil); v != nil {
		t.Errorf("expected nil float slice, got %v", v)
	}
	if v := decodeEmbedding([]byte{1, 2}); v != nil {
		t.Errorf("expected nil for short byte slice (<4 bytes), got %v", v)
	}

	// Normal roundtrip
	orig := []float32{1.5, -3.25, 0.0, 100.5}
	b := encodeEmbedding(orig)
	res := decodeEmbedding(b)
	if len(res) != len(orig) {
		t.Fatalf("expected len %d, got %d", len(orig), len(res))
	}
	for i := range orig {
		if res[i] != orig[i] {
			t.Errorf("dim %d: expected %f, got %f", i, orig[i], res[i])
		}
	}
}

func TestInternal_CosineSimilarity(t *testing.T) {
	// Empty slices
	if s := cosineSimilarity(nil, []float32{1.0}); s != 0 {
		t.Errorf("expected 0 for nil, got %f", s)
	}
	if s := cosineSimilarity([]float32{1.0}, nil); s != 0 {
		t.Errorf("expected 0 for nil, got %f", s)
	}
	// Mismatched length
	if s := cosineSimilarity([]float32{1.0}, []float32{1.0, 2.0}); s != 0 {
		t.Errorf("expected 0 for mismatched length, got %f", s)
	}
	// Zero magnitude
	if s := cosineSimilarity([]float32{0.0, 0.0}, []float32{1.0, 2.0}); s != 0 {
		t.Errorf("expected 0 for zero magnitude, got %f", s)
	}
	if s := cosineSimilarity([]float32{1.0, 2.0}, []float32{0.0, 0.0}); s != 0 {
		t.Errorf("expected 0 for zero magnitude, got %f", s)
	}

	// Identical vectors (cos ~ 1.0)
	v1 := []float32{1.0, 2.0, 3.0}
	if s := cosineSimilarity(v1, v1); math.Abs(float64(s-1.0)) > 1e-5 {
		t.Errorf("expected 1.0 for identical vectors, got %f", s)
	}

	// Orthogonal vectors (cos ~ 0.0)
	vA := []float32{1.0, 0.0}
	vB := []float32{0.0, 1.0}
	if s := cosineSimilarity(vA, vB); math.Abs(float64(s)) > 1e-5 {
		t.Errorf("expected 0.0 for orthogonal vectors, got %f", s)
	}

	// Opposite vectors (cos ~ -1.0)
	vOpp := []float32{-1.0, -2.0, -3.0}
	if s := cosineSimilarity(v1, vOpp); math.Abs(float64(s-(-1.0))) > 1e-5 {
		t.Errorf("expected -1.0 for opposite vectors, got %f", s)
	}
}
