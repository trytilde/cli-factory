package discovery

import (
	"context"
	"math"
	"sort"
	"strings"

	"cli-factory/internal/provider"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type Result struct {
	Kind             string   `json:"kind"`
	Provider         string   `json:"provider"`
	Tool             string   `json:"tool,omitempty"`
	Name             string   `json:"name"`
	Score            float64  `json:"score"`
	ShortDescription string   `json:"short_description"`
	Categories       []string `json:"categories,omitempty"`
	Next             Next     `json:"next"`
}

type Next struct {
	Short string `json:"short"`
	Long  string `json:"long"`
}

func Search(ctx context.Context, r *provider.Registry, embedder Embedder, query string) ([]Result, error) {
	var queryVec []float64
	var err error
	if embedder != nil {
		queryVec, err = embedder.Embed(ctx, query)
		if err != nil {
			return nil, err
		}
	}
	var results []Result
	for _, p := range r.Providers() {
		doc := providerDocument(p)
		providerScore := score(query, doc, queryVec, pseudoVector(doc))
		results = append(results, Result{
			Kind:             "provider",
			Provider:         p.ID(),
			Name:             p.Name(),
			Score:            providerScore,
			ShortDescription: p.ShortDescription(),
			Categories:       p.Categories(),
			Next: Next{
				Short: "factory discover short " + p.ID(),
				Long:  "factory discover long " + p.ID(),
			},
		})
		for _, tool := range p.Tools() {
			toolDoc := toolDocument(p, tool)
			resultScore := score(query, toolDoc, queryVec, pseudoVector(toolDoc))
			results = append(results, Result{
				Kind:             "tool",
				Provider:         p.ID(),
				Tool:             tool.ID(),
				Name:             tool.Name(),
				Score:            resultScore,
				ShortDescription: tool.ShortDescription(),
				Categories:       tool.Categories(),
				Next: Next{
					Short: "factory discover short " + p.ID() + " " + tool.ID(),
					Long:  "factory discover long " + p.ID() + " " + tool.ID(),
				},
			})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Kind < results[j].Kind
		}
		return results[i].Score > results[j].Score
	})
	return results, nil
}

func providerDocument(p provider.Provider) string {
	parts := []string{p.ID(), p.Name(), p.ShortDescription(), p.LongDescription()}
	parts = append(parts, p.Categories()...)
	parts = append(parts, p.Aliases()...)
	for _, t := range p.Tools() {
		parts = append(parts, t.ID(), t.Name(), t.ShortDescription(), t.LongDescription())
	}
	return strings.Join(parts, " ")
}

func toolDocument(p provider.Provider, t provider.Tool) string {
	parts := []string{p.ID(), p.Name(), t.ID(), t.Name(), t.ShortDescription(), t.LongDescription()}
	parts = append(parts, t.Categories()...)
	parts = append(parts, t.Aliases()...)
	return strings.Join(parts, " ")
}

func score(query, doc string, queryVec, docVec []float64) float64 {
	q := strings.ToLower(query)
	d := strings.ToLower(doc)
	var s float64
	for _, token := range strings.Fields(q) {
		if strings.Contains(d, token) {
			s += 0.2
		}
	}
	if strings.Contains(d, q) {
		s += 0.6
	}
	if len(queryVec) > 0 && len(queryVec) == len(docVec) {
		s += cosine(queryVec, docVec)
	}
	return s
}

func pseudoVector(text string) []float64 {
	vec := make([]float64, 16)
	for i, r := range []byte(strings.ToLower(text)) {
		vec[i%len(vec)] += float64(r)
	}
	return vec
}

func cosine(a, b []float64) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
