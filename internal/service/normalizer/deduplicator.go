package normalizer

import (
	"math"
	"regexp"
	"strings"
)

var (
	wordBoundaryRegex = regexp.MustCompile(`[^\w\s]+`)
	indonesianStopwords = map[string]bool{
		"yang": true, "di": true, "ke": true, "dari": true, "ini": true,
		"itu": true, "dan": true, "atau": true, "dengan": true, "untuk": true,
		"pada": true, "adalah": true, "sebagai": true, "akan": true, "oleh": true,
		"juga": true, "bisa": true, "dapat": true, "ada": true, "tidak": true,
		"tersebut": true, "dalam": true, "saat": true, "serta": true, "telah": true,
		"the": true, "and": true, "to": true, "of": true, "in": true, "for": true,
	}
)

type Deduplicator struct{}

func NewDeduplicator() *Deduplicator {
	return &Deduplicator{}
}

// Tokenize converts a text into word term frequency map (TF), stripping stopwords and non-alphanumeric chars.
func (d *Deduplicator) Tokenize(text string) map[string]float64 {
	cleaned := strings.ToLower(wordBoundaryRegex.ReplaceAllString(text, " "))
	words := strings.Fields(cleaned)
	tf := make(map[string]float64)

	for _, w := range words {
		if len(w) <= 2 || indonesianStopwords[w] {
			continue
		}
		tf[w] += 1.0
	}

	return tf
}

// ComputeCosineSimilarity calculates the Cosine Similarity (0.0 to 1.0) between two text documents.
func (d *Deduplicator) ComputeCosineSimilarity(text1, text2 string) float64 {
	tf1 := d.Tokenize(text1)
	tf2 := d.Tokenize(text2)

	if len(tf1) == 0 || len(tf2) == 0 {
		return 0.0
	}

	var dotProduct float64
	var norm1 float64
	var norm2 float64

	for word, count1 := range tf1 {
		norm1 += count1 * count1
		if count2, exists := tf2[word]; exists {
			dotProduct += count1 * count2
		}
	}

	for _, count2 := range tf2 {
		norm2 += count2 * count2
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// ComputeJaccardSimilarity calculates Jaccard Index (0.0 to 1.0) of word sets.
func (d *Deduplicator) ComputeJaccardSimilarity(text1, text2 string) float64 {
	tf1 := d.Tokenize(text1)
	tf2 := d.Tokenize(text2)

	if len(tf1) == 0 || len(tf2) == 0 {
		return 0.0
	}

	set1 := make(map[string]bool)
	for k := range tf1 {
		set1[k] = true
	}

	set2 := make(map[string]bool)
	for k := range tf2 {
		set2[k] = true
	}

	var intersection int
	for k := range set1 {
		if set2[k] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

// IsDuplicate determines if two texts are duplicates based on Cosine Similarity threshold (default 0.85).
func (d *Deduplicator) IsDuplicate(text1, text2 string, threshold float64) bool {
	if threshold <= 0 {
		threshold = 0.85
	}
	sim := d.ComputeCosineSimilarity(text1, text2)
	return sim >= threshold
}

type ArticleItem struct {
	ID      string
	Title   string
	Content string
	URL     string
}

// FilterUniqueArticles removes duplicate news articles from a slice, preserving unique content.
func (d *Deduplicator) FilterUniqueArticles(articles []ArticleItem, threshold float64) []ArticleItem {
	if threshold <= 0 {
		threshold = 0.85
	}

	unique := make([]ArticleItem, 0, len(articles))

	for _, item := range articles {
		isDup := false
		for _, existing := range unique {
			// Check title similarity or content similarity
			titleSim := d.ComputeCosineSimilarity(item.Title, existing.Title)
			contentSim := d.ComputeCosineSimilarity(item.Content, existing.Content)

			if titleSim >= 0.80 || contentSim >= threshold {
				isDup = true
				break
			}
		}

		if !isDup {
			unique = append(unique, item)
		}
	}

	return unique
}
