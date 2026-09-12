package heuristic

import (
	"encoding/json"

	domaincontent "go-api/internal/domain/content"
)

// RulesetConfig is the versioned heuristic calibration payload.
type RulesetConfig struct {
	Version               int
	ShortCircuitOn        []string
	ShortCircuitMinWeight float64
	Checks                map[string]CheckParams
}

type CheckParams struct {
	Enabled    bool               `json:"enabled"`
	Weights    map[string]float64 `json:"weights"`
	Thresholds map[string]float64 `json:"thresholds"`
	Params     map[string]float64 `json:"params"`
}

func (p CheckParams) WeightFor(code string) float64 {
	if p.Weights == nil {
		return 0.2
	}
	if w, ok := p.Weights[code]; ok {
		return w
	}
	return 0.2
}

func (p CheckParams) Threshold(name string) float64 {
	if p.Thresholds == nil {
		return 0
	}
	return p.Thresholds[name]
}

func (p CheckParams) IntParam(name string) int {
	if p.Params == nil {
		return 0
	}
	return int(p.Params[name])
}

func (r RulesetConfig) ParamsFor(checkName string) CheckParams {
	if r.Checks == nil {
		return CheckParams{Enabled: false}
	}
	params, ok := r.Checks[checkName]
	if !ok {
		return CheckParams{Enabled: false}
	}
	return params
}

func (r RulesetConfig) ShouldShortCircuit(signals []domaincontent.Signal) bool {
	if len(r.ShortCircuitOn) == 0 {
		return false
	}
	allowed := make(map[string]struct{}, len(r.ShortCircuitOn))
	for _, code := range r.ShortCircuitOn {
		allowed[code] = struct{}{}
	}
	minWeight := r.ShortCircuitMinWeight
	if minWeight <= 0 {
		minWeight = 0.8
	}
	for _, s := range signals {
		if _, ok := allowed[s.Code]; !ok {
			continue
		}
		if s.Weight >= minWeight {
			return true
		}
	}
	return false
}

type rulesetJSON struct {
	ShortCircuitOn        []string               `json:"shortCircuitOn"`
	ShortCircuitMinWeight float64                `json:"shortCircuitMinWeight"`
	Checks                map[string]CheckParams `json:"checks"`
}

func ParseRulesetParams(version int, raw []byte) (RulesetConfig, error) {
	var parsed rulesetJSON
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return RulesetConfig{}, err
	}
	return RulesetConfig{
		Version:               version,
		ShortCircuitOn:        parsed.ShortCircuitOn,
		ShortCircuitMinWeight: parsed.ShortCircuitMinWeight,
		Checks:                parsed.Checks,
	}, nil
}

func DefaultRuleset() RulesetConfig {
	raw := []byte(`{
      "shortCircuitOn": ["c2pa_generative", "known_hash_match"],
      "shortCircuitMinWeight": 0.8,
      "checks": {
        "exif": {"enabled": true, "weights": {"exif_missing": 0.15, "c2pa_generative": 0.95}},
        "jpeg_quant": {"enabled": true, "weights": {"jpeg_quant_generic": 0.25}},
        "prnu": {"enabled": false},
        "frequency": {"enabled": true, "thresholds": {"frequency_peak_min": 0.35, "frequency_periodicity_max": 0.55}, "weights": {"frequency_artifact": 0.4}},
        "geometry": {"enabled": false},
        "patch_noise": {"enabled": true, "params": {"patch_grid_size": 8, "patch_outlier_max_count": 6}, "thresholds": {"patch_variance_stddev": 2.0}, "weights": {"patch_noise_inconsistent": 0.35}},
        "known_hash": {"enabled": true, "params": {"hash_max_distance": 8}, "weights": {"known_hash_match": 0.9}}
      }
    }`)
	cfg, err := ParseRulesetParams(0, raw)
	if err != nil {
		return RulesetConfig{Version: 0, Checks: map[string]CheckParams{}}
	}
	return cfg
}
