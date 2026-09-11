package normalizer

import (
	"testing"
)

func TestDeduplicator_ComputeCosineSimilarity(t *testing.T) {
	dedup := NewDeduplicator()

	text1 := "PT Telkom Indonesia mengalokasikan dana CSR sebesar Rp 25 Miliar untuk beasiswa digital di Papua dan NTT."
	text2 := "PT Telkom Indonesia mengalokasikan anggaran CSR senilai Rp 25 Miliar bagi beasiswa digital daerah Papua dan NTT."
	text3 := "PT Bank Mandiri Tbk menggelar operasi pasar murah dan bantuan sembako bagi masyarakat terdampak banjir."

	sim12 := dedup.ComputeCosineSimilarity(text1, text2)
	sim13 := dedup.ComputeCosineSimilarity(text1, text3)

	if sim12 < 0.70 {
		t.Errorf("Expected high similarity between text1 and text2, got %f", sim12)
	}

	if sim13 > 0.40 {
		t.Errorf("Expected low similarity between text1 and text3, got %f", sim13)
	}

	if !dedup.IsDuplicate(text1, text2, 0.70) {
		t.Errorf("Expected text1 and text2 to be flagged as duplicates at 0.70 threshold")
	}

	if dedup.IsDuplicate(text1, text3, 0.70) {
		t.Errorf("Expected text1 and text3 NOT to be flagged as duplicates")
	}
}

func TestDeduplicator_FilterUniqueArticles(t *testing.T) {
	dedup := NewDeduplicator()

	articles := []ArticleItem{
		{ID: "1", Title: "Telkom Alokasikan Beasiswa Digital", Content: "Telkom memberikan beasiswa digital Rp 25 Miliar bagi mahasiswa."},
		{ID: "2", Title: "Telkom Beri Beasiswa Digital Rp 25M", Content: "Telkom memberikan beasiswa digital senilai Rp 25 Miliar bagi mahasiswa Indonesia."},
		{ID: "3", Title: "PLN Bangun Pembangkit Listrik Surya", Content: "PLN meresmikan PLTS baru di Sumbawa NTT."},
	}

	unique := dedup.FilterUniqueArticles(articles, 0.75)
	if len(unique) != 2 {
		t.Errorf("Expected 2 unique articles after deduplication, got %d", len(unique))
	}
}
