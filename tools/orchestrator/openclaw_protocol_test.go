package orchestrator

import (
	"encoding/json"
	"testing"
)

func TestOpenClawRequestMarshal(t *testing.T) {
	payload := EvaluateClaimPayload{
		ClaimID:        "claim-1",
		ClaimText:      "Water boils at 100 degrees Celsius",
		Domain:         "physics",
		Category:       "empirical",
		DomainDensity:  42,
		StratumCeiling: 990_000,
		LogicZoneCap:   700_000,
		SupportingFacts: []FactSummary{
			{ID: "f1", Text: "H2O phase transition", Confidence: 800_000, Maturity: "canonical", RefType: "supports"},
		},
		CounterFacts: []CounterFactSummary{
			{ID: "cf1", NegationType: "qualification", Confidence: 400_000},
		},
	}

	payloadData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := OpenClawRequest{
		Type:    OCmdEvaluateClaim,
		Payload: payloadData,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	// Unmarshal back
	var decoded OpenClawRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	if decoded.Type != OCmdEvaluateClaim {
		t.Fatalf("expected type %s, got %s", OCmdEvaluateClaim, decoded.Type)
	}

	var decodedPayload EvaluateClaimPayload
	if err := json.Unmarshal(decoded.Payload, &decodedPayload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if decodedPayload.ClaimID != "claim-1" {
		t.Fatalf("expected claim-1, got %s", decodedPayload.ClaimID)
	}
	if decodedPayload.StratumCeiling != 990_000 {
		t.Fatalf("expected 990000, got %d", decodedPayload.StratumCeiling)
	}
	if len(decodedPayload.SupportingFacts) != 1 {
		t.Fatalf("expected 1 supporting fact, got %d", len(decodedPayload.SupportingFacts))
	}
	if len(decodedPayload.CounterFacts) != 1 {
		t.Fatalf("expected 1 counter fact, got %d", len(decodedPayload.CounterFacts))
	}
}

func TestOpenClawResponseVerdictParsing(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		verdict  string
		conf     uint32
		hasError bool
	}{
		{
			name:    "accept verdict",
			json:    `{"type":"evaluate_claim","verdict":"accept","confidence":750000,"reasoning":"Strong evidence"}`,
			verdict: "accept",
			conf:    750_000,
		},
		{
			name:    "reject verdict",
			json:    `{"type":"evaluate_claim","verdict":"reject","confidence":200000,"reasoning":"Contradicted"}`,
			verdict: "reject",
			conf:    200_000,
		},
		{
			name:    "abstain verdict",
			json:    `{"type":"evaluate_claim","verdict":"abstain","confidence":400000}`,
			verdict: "abstain",
			conf:    400_000,
		},
		{
			name:     "error response",
			json:     `{"type":"evaluate_claim","error":"session timeout"}`,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var resp OpenClawResponse
			if err := json.Unmarshal([]byte(tt.json), &resp); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			if tt.hasError {
				if resp.Error == "" {
					t.Fatal("expected error field")
				}
				return
			}

			if resp.Verdict != tt.verdict {
				t.Fatalf("expected verdict %s, got %s", tt.verdict, resp.Verdict)
			}
			if resp.Confidence != tt.conf {
				t.Fatalf("expected confidence %d, got %d", tt.conf, resp.Confidence)
			}
		})
	}
}

func TestOpenClawResponseGapsParsing(t *testing.T) {
	raw := `{
		"type": "identify_gaps",
		"gaps": [
			{"type": "low_density", "domain": "biology", "description": "few facts", "priority": 8000},
			{"type": "decaying", "domain": "biology", "description": "old facts", "priority": 5000}
		]
	}`

	var resp OpenClawResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Gaps) != 2 {
		t.Fatalf("expected 2 gaps, got %d", len(resp.Gaps))
	}
	if resp.Gaps[0].Type != "low_density" {
		t.Fatalf("expected low_density, got %s", resp.Gaps[0].Type)
	}
	if resp.Gaps[0].Priority != 8000 {
		t.Fatalf("expected priority 8000, got %d", resp.Gaps[0].Priority)
	}
}

func TestOpenClawResponseClaimsParsing(t *testing.T) {
	raw := `{
		"type": "generate_claim",
		"claims": [
			{"text": "Mitochondria are the powerhouse of the cell", "domain": "biology", "category": "empirical", "stake": 5000000}
		]
	}`

	var resp OpenClawResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Claims) != 1 {
		t.Fatalf("expected 1 claim, got %d", len(resp.Claims))
	}
	if resp.Claims[0].Category != "empirical" {
		t.Fatalf("expected empirical, got %s", resp.Claims[0].Category)
	}
	if resp.Claims[0].Stake != 5_000_000 {
		t.Fatalf("expected stake 5000000, got %d", resp.Claims[0].Stake)
	}
}

func TestOpenClawAllCommandTypesRoundTrip(t *testing.T) {
	commandTypes := []string{
		OCmdEvaluateClaim,
		OCmdIdentifyGaps,
		OCmdAnalyzeReferences,
		OCmdGenerateClaim,
	}

	for _, ct := range commandTypes {
		req := OpenClawRequest{
			Type:    ct,
			Payload: json.RawMessage(`{"test": true}`),
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("marshal %s: %v", ct, err)
		}

		var decoded OpenClawRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal %s: %v", ct, err)
		}

		if decoded.Type != ct {
			t.Fatalf("expected %s, got %s", ct, decoded.Type)
		}
	}
}

func TestIdentifyGapsPayloadRoundTrip(t *testing.T) {
	payload := IdentifyGapsPayload{
		Domain:    "physics",
		FactCount: 42,
		TopFacts: []FactSummary{
			{ID: "f1", Text: "E=mc²", Confidence: 900_000, Maturity: "canonical"},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded IdentifyGapsPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Domain != "physics" {
		t.Fatalf("expected physics, got %s", decoded.Domain)
	}
	if decoded.FactCount != 42 {
		t.Fatalf("expected 42, got %d", decoded.FactCount)
	}
}

func TestGenerateClaimPayloadRoundTrip(t *testing.T) {
	payload := GenerateClaimPayload{
		Domain: "biology",
		Gaps: []GapSuggestion{
			{Type: "low_density", Domain: "biology", Description: "need more cell biology facts", Priority: 7000},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded GenerateClaimPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Domain != "biology" {
		t.Fatalf("expected biology, got %s", decoded.Domain)
	}
	if len(decoded.Gaps) != 1 {
		t.Fatalf("expected 1 gap, got %d", len(decoded.Gaps))
	}
}

func TestAnalyzeReferencesPayloadRoundTrip(t *testing.T) {
	payload := AnalyzeReferencesPayload{
		FactID:    "fact-1",
		NodeCount: 5,
		EdgeCount: 4,
		Depth:     3,
		TopFacts: []FactSummary{
			{ID: "fact-1", Text: "root fact", Confidence: 800_000},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded AnalyzeReferencesPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.FactID != "fact-1" {
		t.Fatalf("expected fact-1, got %s", decoded.FactID)
	}
	if decoded.NodeCount != 5 {
		t.Fatalf("expected 5 nodes, got %d", decoded.NodeCount)
	}
}
