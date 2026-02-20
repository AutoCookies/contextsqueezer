package metrics

type StageMetrics struct {
	IngestMS             int64  `json:"ingest_ms"`
	SegmentationMS       int64  `json:"segmentation_ms"`
	BoilerplateMS        int64  `json:"boilerplate_ms"`
	TokenizationMS       int64  `json:"tokenization_ms"`
	CandidateFilterMS    int64  `json:"candidate_filter_ms"`
	SimilarityMS         int64  `json:"similarity_ms"`
	PruneMS              int64  `json:"prune_ms"`
	AssemblyMS           int64  `json:"assembly_ms"`
	CrossChunkRegistryMS int64  `json:"cross_chunk_registry_ms"`
	BudgetLoopMS         int64  `json:"budget_loop_ms"`
	SimilarityCandidates uint64 `json:"similarity_candidates_checked"`
	SimilarityPairs      uint64 `json:"similarity_pairs_compared"`
	TokensParsed         uint64 `json:"tokens_parsed"`
	SentencesTotal       uint64 `json:"sentences_total"`
	PeakMemoryEstimateB  int64  `json:"peak_memory_estimate_bytes"`
}
