package content

// Signal is an explainable analysis finding from a detector.
type Signal struct {
	Type        string  `json:"type"`
	Code        string  `json:"code"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
}

// Verdict is the aggregated outcome of all detectors.
type Verdict struct {
	Label      Label
	Confidence float64
	Signals    []Signal
}
