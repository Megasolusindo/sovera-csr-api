package ai

import (
	"testing"
)

func TestComputeFlexibleIdempotencyKey(t *testing.T) {
	key1 := ComputeFlexibleIdempotencyKey("src_101", "CLAIM_EXTRACTION_FUNDING", "v1.2.0", "p1.0")
	key2 := ComputeFlexibleIdempotencyKey("src_101", "CLAIM_EXTRACTION_FUNDING", "v1.2.0", "p1.0")
	key3 := ComputeFlexibleIdempotencyKey("src_101", "CLAIM_EXTRACTION_FUNDING", "v1.2.1", "p1.0") // different pipeline version

	if key1 != key2 {
		t.Errorf("idempotency key must be deterministic for identical inputs")
	}

	if key1 == key3 {
		t.Errorf("idempotency key must change when pipeline version changes (§18 flexible idempotency key)")
	}
}

func TestPressReleaseTracker_SyndicationMatching(t *testing.T) {
	text1 := "JAKARTA, 15 September 2026 - PT Bank Syariah Indonesia Tbk (BSI) menyalurkan dana CSR sebesar Rp 153 Miliar untuk program pemberdayaan ekonomi masyarakat melalui LAZ."
	text2 := "Semarang, 16 Sep 2026 (DETIK) -- PT Bank Syariah Indonesia Tbk (BSI) menyalurkan dana CSR sebesar Rp 153 Miliar untuk program pemberdayaan ekonomi masyarakat melalui LAZ."

	hash1 := ComputePressReleaseHash(text1)
	hash2 := ComputePressReleaseHash(text2)

	if hash1 != hash2 {
		t.Errorf("expected press release hashes to match for syndicated articles, got %s vs %s", hash1, hash2)
	}

	existingHashes := map[string]string{
		hash1: "https://bsi.co.id/press-release-csr-153b",
	}

	syndication := TrackSyndication(existingHashes, text2, "https://detik.com/news/bsi-csr")
	if !syndication.IsSyndicatedCopy {
		t.Errorf("expected article 2 to be marked as syndicated copy")
	}
	if syndication.PrimaryOrigin != "https://bsi.co.id/press-release-csr-153b" {
		t.Errorf("expected primary origin to be original wire URL, got %s", syndication.PrimaryOrigin)
	}
}
