package cache

import (
	"hash/fnv"
	"math"
	"strings"
)

const embeddingDims = 384

// Embedder converts text into a fixed-size float32 vector.
// This pure-Go implementation uses FNV hashing over word n-grams.
// It captures lexical similarity but not semantic similarity.
// Replace with ONNX all-MiniLM-L6-v2 for production semantic search.
type Embedder struct{}

func NewEmbedder() *Embedder {
	return &Embedder{}
}

// Embed converts text to a normalized 384-dimensional float32 vector.
func (e *Embedder) Embed(text string) []float32 {
	vec := make([]float32, embeddingDims)

	text = strings.ToLower(strings.TrimSpace(text))
	tokens := strings.Fields(text)

	// Unigrams
	for _, token := range tokens {
		idx := hashToIndex(token)
		vec[idx] += 1.0
	}

	// Bigrams — capture some phrase-level information
	for i := 0; i < len(tokens)-1; i++ {
		bigram := tokens[i] + "_" + tokens[i+1]
		idx := hashToIndex(bigram)
		vec[idx] += 0.5
	}

	return normalize(vec)
}

// hashToIndex maps a string to a bucket in [0, embeddingDims).
func hashToIndex(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32()) % embeddingDims
}

// normalize scales the vector to unit length (required for cosine similarity).
func normalize(vec []float32) []float32 {
	var sum float64
	for _, v := range vec {
		sum += float64(v * v)
	}
	if sum == 0 {
		return vec
	}
	norm := float32(math.Sqrt(sum))
	for i := range vec {
		vec[i] /= norm
	}
	return vec
}
