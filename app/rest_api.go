package app

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	sdk "github.com/cosmos/cosmos-sdk/types"

	capturechallengetypes "github.com/legible-money/legible/x/capture_challenge/types"
	claimingpottypes "github.com/legible-money/legible/x/claiming_pot/types"
	emergencytypes "github.com/legible-money/legible/x/emergency/types"
	hometypes "github.com/legible-money/legible/x/home/types"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
	ontologytypes "github.com/legible-money/legible/x/ontology/types"
	researchtypes "github.com/legible-money/legible/x/research/types"
	legiblestakingtypes "github.com/legible-money/legible/x/staking/types"
)

// registerCustomRESTRoutes registers REST API routes for custom Legible Money
// modules. These modules use hand-written types (not proto-generated), so they
// cannot use the standard grpc-gateway code generation. Instead, routes call
// keepers directly via the latest committed state.
func registerCustomRESTRoutes(app *LegibleApp, router *mux.Router) {
	registerKnowledgeRoutes(app, router)
	registerLegibleAuthRoutes(app, router)
	registerLegibleStakingRoutes(app, router)
	registerHomeRoutes(app, router)
	registerDiscoveryRoutes(app, router)
	registerEmergencyRoutes(app, router)
	registerLegibleGovRoutes(app, router)
	registerVestingRewardsRoutes(app, router)
	registerTokensRoutes(app, router)
	registerPartnershipsRoutes(app, router)
	registerBillingRoutes(app, router)
	registerDisputesRoutes(app, router)
	registerEvidenceMgmtRoutes(app, router)
	registerClaimingPotRoutes(app, router)
	registerChannelsRoutes(app, router)
	registerTreeRoutes(app, router)
	registerBVMRoutes(app, router)
	registerQualificationRoutes(app, router)
	registerCaptureChallengeRoutes(app, router)
	registerCaptureDefenseRoutes(app, router)
	registerOntologyRoutes(app, router)
	registerResearchRoutes(app, router)
	registerScheduleRoutes(app, router)
	registerIBCRateLimitRoutes(app, router)
	registerICAAuthRoutes(app, router)
	registerAutopoiesisRoutes(app, router)
	registerAlignmentRoutes(app, router)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}

func queryCtx(app *LegibleApp) (sdk.Context, error) {
	return app.CreateQueryContext(0, false)
}

func parseUint32(r *http.Request, key string, def uint32) uint32 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return def
	}
	return uint32(n)
}

// --- x/knowledge routes (12 endpoints) ---

func registerKnowledgeRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/knowledge/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.KnowledgeKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/knowledge/v1/fact/{fact_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		fact, found := app.KnowledgeKeeper.GetFact(ctx, mux.Vars(r)["fact_id"])
		if !found {
			writeError(w, http.StatusNotFound, "fact not found")
			return
		}
		writeJSON(w, map[string]interface{}{"fact": fact})
	}).Methods("GET")

	router.HandleFunc("/legible/knowledge/v1/facts", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		limit := parseUint32(r, "limit", 100)
		if limit > 100 {
			limit = 100
		}
		offset := parseUint32(r, "offset", 0)

		var facts []*knowledgetypes.Fact
		var total uint64
		var skipped uint32
		app.KnowledgeKeeper.IterateFacts(ctx, func(f *knowledgetypes.Fact) bool {
			total++
			if skipped < offset {
				skipped++
				return false
			}
			if uint32(len(facts)) >= limit {
				return false
			}
			facts = append(facts, f)
			return false
		})
		writeJSON(w, map[string]interface{}{"facts": facts, "total": total})
	}).Methods("GET")

	router.HandleFunc("/legible/knowledge/v1/facts/domain/{domain}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"facts": app.KnowledgeKeeper.GetFactsByDomain(ctx, mux.Vars(r)["domain"])})
	}).Methods("GET")

	router.HandleFunc("/legible/knowledge/v1/claim/{claim_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		claim, found := app.KnowledgeKeeper.GetClaim(ctx, mux.Vars(r)["claim_id"])
		if !found {
			writeError(w, http.StatusNotFound, "claim not found")
			return
		}
		writeJSON(w, map[string]interface{}{"claim": claim})
	}).Methods("GET")

	router.HandleFunc("/legible/knowledge/v1/round/{round_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		round, found := app.KnowledgeKeeper.GetVerificationRound(ctx, mux.Vars(r)["round_id"])
		if !found {
			writeError(w, http.StatusNotFound, "round not found")
			return
		}
		writeJSON(w, map[string]interface{}{"round": round})
	}).Methods("GET")

	router.HandleFunc("/legible/knowledge/v1/rounds/active", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"rounds": app.KnowledgeKeeper.GetActiveRounds(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/knowledge/v1/domain/{domain_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		domain, found := app.KnowledgeKeeper.GetDomainFromState(ctx, mux.Vars(r)["domain_id"])
		if !found {
			writeError(w, http.StatusNotFound, "domain not found")
			return
		}
		writeJSON(w, map[string]interface{}{"domain": domain})
	}).Methods("GET")

	router.HandleFunc("/legible/knowledge/v1/counter_fact/{counter_fact_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		cf, found := app.KnowledgeKeeper.GetCounterFact(ctx, mux.Vars(r)["counter_fact_id"])
		if !found {
			writeError(w, http.StatusNotFound, "counter-fact not found")
			return
		}
		writeJSON(w, map[string]interface{}{"counter_fact": cf})
	}).Methods("GET")

	router.HandleFunc("/legible/knowledge/v1/counter_facts/fact/{fact_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"counter_facts": app.KnowledgeKeeper.GetCounterFactsByFact(ctx, mux.Vars(r)["fact_id"])})
	}).Methods("GET")

	router.HandleFunc("/legible/knowledge/v1/counter_facts/domain/{domain}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"counter_facts": app.KnowledgeKeeper.GetCounterFactsByDomain(ctx, mux.Vars(r)["domain"])})
	}).Methods("GET")

	router.HandleFunc("/legible/knowledge/v1/bidirectional_confidence/{fact_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		factID := mux.Vars(r)["fact_id"]
		if _, found := app.KnowledgeKeeper.GetFact(ctx, factID); !found {
			writeError(w, http.StatusNotFound, "fact not found")
			return
		}
		writeJSON(w, map[string]interface{}{"bidirectional_confidence": app.KnowledgeKeeper.GetBidirectionalConfidence(ctx, factID)})
	}).Methods("GET")
}

// --- x/auth (legible_auth) routes (4 endpoints) ---

func registerLegibleAuthRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/auth/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.LegibleAuthKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/auth/v1/account/{address}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		account, found := app.LegibleAuthKeeper.GetAccount(ctx, mux.Vars(r)["address"])
		if !found {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}
		writeJSON(w, map[string]interface{}{"account": account})
	}).Methods("GET")

	router.HandleFunc("/legible/auth/v1/account_by_did/{did}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		account, found := app.LegibleAuthKeeper.GetAccountByDID(ctx, mux.Vars(r)["did"])
		if !found {
			writeError(w, http.StatusNotFound, "account not found")
			return
		}
		writeJSON(w, map[string]interface{}{"account": account})
	}).Methods("GET")

	router.HandleFunc("/legible/auth/v1/session_keys/{owner}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"session_keys": app.LegibleAuthKeeper.GetSessionKeysForOwner(ctx, mux.Vars(r)["owner"])})
	}).Methods("GET")
}

// --- x/staking (legible_staking) routes (6 endpoints) ---

func registerLegibleStakingRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/staking/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{
			"params":       app.LegibleStakingKeeper.GetParams(ctx),
			"tier_configs": app.LegibleStakingKeeper.GetAllTierConfigs(ctx),
		})
	}).Methods("GET")

	router.HandleFunc("/legible/staking/v1/validator/{operator_address}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		v, found := app.LegibleStakingKeeper.GetValidator(ctx, mux.Vars(r)["operator_address"])
		if !found {
			writeError(w, http.StatusNotFound, "validator not found")
			return
		}
		writeJSON(w, map[string]interface{}{"validator": v})
	}).Methods("GET")

	router.HandleFunc("/legible/staking/v1/validators", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		limit := parseUint32(r, "limit", 100)
		if limit > 100 {
			limit = 100
		}
		offset := parseUint32(r, "offset", 0)

		var all []*legiblestakingtypes.LegibleValidator
		app.LegibleStakingKeeper.IterateValidators(ctx, func(v *legiblestakingtypes.LegibleValidator) bool {
			all = append(all, v)
			return false
		})

		total := uint32(len(all))
		start := offset
		if start >= total {
			writeJSON(w, map[string]interface{}{"validators": []*legiblestakingtypes.LegibleValidator{}, "total": total})
			return
		}
		end := start + limit
		if end > total {
			end = total
		}
		writeJSON(w, map[string]interface{}{"validators": all[start:end], "total": total})
	}).Methods("GET")

	router.HandleFunc("/legible/staking/v1/delegation/{delegator_address}/{validator_address}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		vars := mux.Vars(r)
		d, found := app.LegibleStakingKeeper.GetDelegation(ctx, vars["delegator_address"], vars["validator_address"])
		if !found {
			writeError(w, http.StatusNotFound, "delegation not found")
			return
		}
		writeJSON(w, map[string]interface{}{"delegation": d})
	}).Methods("GET")

	router.HandleFunc("/legible/staking/v1/delegations/{validator_address}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"delegations": app.LegibleStakingKeeper.GetDelegationsForValidator(ctx, mux.Vars(r)["validator_address"])})
	}).Methods("GET")

	router.HandleFunc("/legible/staking/v1/unbondings/{delegator_address}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"unbondings": app.LegibleStakingKeeper.GetUnbondingsForDelegator(ctx, mux.Vars(r)["delegator_address"])})
	}).Methods("GET")
}

// --- x/home routes (5 endpoints) ---

func registerHomeRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/home/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.HomeKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/home/v1/home/{did}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		home, found := app.HomeKeeper.GetHome(ctx, mux.Vars(r)["did"])
		if !found {
			writeJSON(w, map[string]interface{}{"home": nil, "found": false})
			return
		}
		writeJSON(w, map[string]interface{}{"home": home, "found": true})
	}).Methods("GET")

	router.HandleFunc("/legible/home/v1/homes", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		limit := parseUint32(r, "limit", 100)
		offset := parseUint32(r, "offset", 0)

		var all []*hometypes.AgentHome
		app.HomeKeeper.IterateHomes(ctx, func(home *hometypes.AgentHome) bool {
			all = append(all, home)
			return false
		})

		total := uint64(len(all))
		start := int(offset)
		end := start + int(limit)
		if start > len(all) {
			start = len(all)
		}
		if end > len(all) {
			end = len(all)
		}
		writeJSON(w, map[string]interface{}{"homes": all[start:end], "total": total})
	}).Methods("GET")

	router.HandleFunc("/legible/home/v1/sessions/{home_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		var sessions []*hometypes.ActiveSession
		app.HomeKeeper.IterateSessions(ctx, mux.Vars(r)["home_id"], func(s *hometypes.ActiveSession) bool {
			sessions = append(sessions, s)
			return false
		})
		writeJSON(w, map[string]interface{}{"sessions": sessions})
	}).Methods("GET")

	router.HandleFunc("/legible/home/v1/alerts/{home_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		var alerts []*hometypes.Alert
		app.HomeKeeper.IterateAlerts(ctx, mux.Vars(r)["home_id"], func(a *hometypes.Alert) bool {
			alerts = append(alerts, a)
			return false
		})
		writeJSON(w, map[string]interface{}{"alerts": alerts})
	}).Methods("GET")

	router.HandleFunc("/legible/home/v1/maturity/{home_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		homeId := mux.Vars(r)["home_id"]
		stage := app.HomeKeeper.GetHomeMaturity(ctx, homeId)
		multiplier := app.HomeKeeper.GetHomeRewardMultiplierBps(ctx, homeId)
		writeJSON(w, map[string]interface{}{
			"home_id":               homeId,
			"maturity_stage":        stage,
			"reward_multiplier_bps": multiplier,
		})
	}).Methods("GET")

	router.HandleFunc("/legible/home/v1/epistemic-stats/{home_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		homeId := mux.Vars(r)["home_id"]
		stats := app.HomeKeeper.GetEpistemicStats(ctx, homeId)
		writeJSON(w, map[string]interface{}{
			"home_id":         homeId,
			"stats":           stats,
			"epistemic_score": stats.EpistemicScore(),
			"accuracy_rate":   stats.AccuracyRate(),
		})
	}).Methods("GET")
}

// --- x/discovery routes (4 endpoints) ---

func registerDiscoveryRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/discovery/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.DiscoveryKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/discovery/v1/agent/{address}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		profile, found := app.DiscoveryKeeper.GetProfile(ctx, mux.Vars(r)["address"])
		if !found {
			writeError(w, http.StatusNotFound, "agent not found")
			return
		}
		writeJSON(w, map[string]interface{}{"profile": profile})
	}).Methods("GET")

	router.HandleFunc("/legible/discovery/v1/agents/domain/{domain}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"profiles": app.DiscoveryKeeper.GetProfilesByDomain(ctx, mux.Vars(r)["domain"])})
	}).Methods("GET")

	router.HandleFunc("/legible/discovery/v1/agents/capability/{capability_type}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"profiles": app.DiscoveryKeeper.GetProfilesByCapability(ctx, mux.Vars(r)["capability_type"])})
	}).Methods("GET")
}

// --- x/emergency routes (4 endpoints) ---

func registerEmergencyRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/emergency/v1/status", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{
			"status":                  app.EmergencyKeeper.GetEmergencyStatus(ctx),
			"is_halted":               app.EmergencyKeeper.IsHalted(ctx),
			"active_halt_ceremony_id": app.EmergencyKeeper.GetActiveHaltCeremonyId(ctx),
		})
	}).Methods("GET")

	router.HandleFunc("/legible/emergency/v1/active_ceremony", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		ceremony, found := app.EmergencyKeeper.GetActiveCeremony(ctx)
		if !found {
			writeJSON(w, map[string]interface{}{"ceremony": nil, "found": false})
			return
		}
		writeJSON(w, map[string]interface{}{"ceremony": ceremony, "found": true})
	}).Methods("GET")

	router.HandleFunc("/legible/emergency/v1/completed_ceremonies", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		limit := parseUint32(r, "limit", 100)
		if limit > 100 {
			limit = 100
		}
		offset := parseUint32(r, "offset", 0)

		var completed []*emergencytypes.EmergencyCeremony
		app.EmergencyKeeper.IterateCeremonies(ctx, func(c *emergencytypes.EmergencyCeremony) bool {
			completed = append(completed, c)
			return false
		})

		total := uint64(len(completed))
		start := int(offset)
		end := start + int(limit)
		if start > len(completed) {
			start = len(completed)
		}
		if end > len(completed) {
			end = len(completed)
		}
		writeJSON(w, map[string]interface{}{"ceremonies": completed[start:end], "total": total})
	}).Methods("GET")

	router.HandleFunc("/legible/emergency/v1/audit_log", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		limit := parseUint32(r, "limit", 100)
		if limit > 100 {
			limit = 100
		}
		offset := parseUint32(r, "offset", 0)

		entries := app.EmergencyKeeper.GetAuditLog(ctx)
		total := uint64(len(entries))

		start := int(offset)
		end := start + int(limit)
		if start > len(entries) {
			start = len(entries)
		}
		if end > len(entries) {
			end = len(entries)
		}
		writeJSON(w, map[string]interface{}{"entries": entries[start:end], "total": total})
	}).Methods("GET")
}

// --- x/gov (legible_gov) routes (4 endpoints) ---

func registerLegibleGovRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/gov/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.LegibleGovKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/gov/v1/lip/{lip_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		lipID := mux.Vars(r)["lip_id"]
		lip, found := app.LegibleGovKeeper.GetLIP(ctx, lipID)
		if !found {
			writeError(w, http.StatusNotFound, "LIP not found")
			return
		}
		writeJSON(w, map[string]interface{}{"lip": lip, "votes": app.LegibleGovKeeper.GetVotesForProposal(ctx, lipID)})
	}).Methods("GET")

	router.HandleFunc("/legible/gov/v1/lips", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		limit := parseUint32(r, "limit", 100)
		if limit > 100 {
			limit = 100
		}
		offset := parseUint32(r, "offset", 0)

		allLIPs := app.LegibleGovKeeper.GetAllLIPs(ctx)
		total := uint32(len(allLIPs))
		start := offset
		if start >= total {
			writeJSON(w, map[string]interface{}{"lips": allLIPs[:0], "total": total})
			return
		}
		end := start + limit
		if end > total {
			end = total
		}
		writeJSON(w, map[string]interface{}{"lips": allLIPs[start:end], "total": total})
	}).Methods("GET")

	router.HandleFunc("/legible/gov/v1/disbursement/{disbursement_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		disbID := mux.Vars(r)["disbursement_id"]
		disb, found := app.LegibleGovKeeper.GetDisbursement(ctx, disbID)
		if !found {
			writeError(w, http.StatusNotFound, "disbursement not found")
			return
		}
		writeJSON(w, map[string]interface{}{"disbursement": disb, "votes": app.LegibleGovKeeper.GetVotesForProposal(ctx, disbID)})
	}).Methods("GET")
}

// --- x/vesting_rewards routes (5 endpoints) ---

func registerVestingRewardsRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/vesting_rewards/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{
			"params":           app.VestingRewardsKeeper.GetParams(ctx),
			"category_configs": app.VestingRewardsKeeper.GetAllCategoryConfigs(ctx),
		})
	}).Methods("GET")

	router.HandleFunc("/legible/vesting_rewards/v1/schedule/{vesting_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		schedule, found := app.VestingRewardsKeeper.GetVestingSchedule(ctx, mux.Vars(r)["vesting_id"])
		if !found {
			writeError(w, http.StatusNotFound, "schedule not found")
			return
		}
		writeJSON(w, map[string]interface{}{"schedule": schedule})
	}).Methods("GET")

	router.HandleFunc("/legible/vesting_rewards/v1/schedules/{recipient}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"schedules": app.VestingRewardsKeeper.GetVestingSchedulesByRecipient(ctx, mux.Vars(r)["recipient"])})
	}).Methods("GET")

	router.HandleFunc("/legible/vesting_rewards/v1/active_schedules", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		all := app.VestingRewardsKeeper.GetAllActiveVestingSchedules(ctx)
		limit := parseUint32(r, "limit", 100)
		if limit > 100 {
			limit = 100
		}
		offset := parseUint32(r, "offset", 0)

		total := uint32(len(all))
		start := offset
		if start >= total {
			writeJSON(w, map[string]interface{}{"schedules": all[:0], "total": total})
			return
		}
		end := start + limit
		if end > total {
			end = total
		}
		writeJSON(w, map[string]interface{}{"schedules": all[start:end], "total": total})
	}).Methods("GET")

	router.HandleFunc("/legible/vesting_rewards/v1/block_reward/{block_height}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		h, _ := strconv.ParseUint(mux.Vars(r)["block_height"], 10, 64)
		dist, found := app.VestingRewardsKeeper.GetBlockRewardDistribution(ctx, h)
		writeJSON(w, map[string]interface{}{"distribution": dist, "found": found})
	}).Methods("GET")
}

// --- x/tokens routes (1 endpoint) ---

func registerTokensRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/tokens/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.TokensKeeper.GetParams(ctx)})
	}).Methods("GET")
}

// --- x/partnerships routes (3 endpoints) ---

func registerPartnershipsRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/partnerships/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.PartnershipsKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/partnerships/v1/partnership/{partnership_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		p, found := app.PartnershipsKeeper.GetPartnership(ctx, mux.Vars(r)["partnership_id"])
		if !found {
			writeError(w, http.StatusNotFound, "partnership not found")
			return
		}
		writeJSON(w, map[string]interface{}{"partnership": p})
	}).Methods("GET")

	router.HandleFunc("/legible/partnerships/v1/partnerships/{participant}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"partnerships": app.PartnershipsKeeper.GetPartnershipsByParticipant(ctx, mux.Vars(r)["participant"])})
	}).Methods("GET")
}

// --- x/billing routes (3 endpoints) ---

func registerBillingRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/billing/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.BillingKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/billing/v1/provider/{address}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		provider, found := app.BillingKeeper.GetProvider(ctx, mux.Vars(r)["address"])
		if !found {
			writeError(w, http.StatusNotFound, "provider not found")
			return
		}
		writeJSON(w, map[string]interface{}{"provider": provider})
	}).Methods("GET")

	router.HandleFunc("/legible/billing/v1/quote/{quote_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		qid, _ := strconv.ParseUint(mux.Vars(r)["quote_id"], 10, 64)
		quote, found := app.BillingKeeper.GetQuote(ctx, qid)
		if !found {
			writeError(w, http.StatusNotFound, "quote not found")
			return
		}
		writeJSON(w, map[string]interface{}{"quote": quote})
	}).Methods("GET")
}

// --- x/disputes routes (6 endpoints) ---

func registerDisputesRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/disputes/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.DisputesKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/disputes/v1/dispute/{dispute_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		d, found := app.DisputesKeeper.GetDispute(ctx, mux.Vars(r)["dispute_id"])
		if !found {
			writeError(w, http.StatusNotFound, "dispute not found")
			return
		}
		writeJSON(w, map[string]interface{}{"dispute": d})
	}).Methods("GET")

	router.HandleFunc("/legible/disputes/v1/disputes/fact/{fact_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"disputes": app.DisputesKeeper.GetDisputesByFact(ctx, mux.Vars(r)["fact_id"])})
	}).Methods("GET")

	router.HandleFunc("/legible/disputes/v1/disputes/active", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"disputes": app.DisputesKeeper.GetActiveDisputes(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/disputes/v1/evidence/{dispute_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"evidence": app.DisputesKeeper.GetEvidenceByDispute(ctx, mux.Vars(r)["dispute_id"])})
	}).Methods("GET")

	router.HandleFunc("/legible/disputes/v1/votes/{dispute_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"votes": app.DisputesKeeper.GetVotes(ctx, mux.Vars(r)["dispute_id"])})
	}).Methods("GET")
}

// --- x/evidence_mgmt routes (8 endpoints) ---

func registerEvidenceMgmtRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/evidence_mgmt/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.EvidenceMgmtKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/evidence_mgmt/v1/evidence/{evidence_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		ev, found := app.EvidenceMgmtKeeper.GetEvidence(ctx, mux.Vars(r)["evidence_id"])
		if !found {
			writeError(w, http.StatusNotFound, "evidence not found")
			return
		}
		writeJSON(w, map[string]interface{}{"evidence": ev})
	}).Methods("GET")

	router.HandleFunc("/legible/evidence_mgmt/v1/evidence/claim/{claim_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"evidence": app.EvidenceMgmtKeeper.GetEvidenceByClaimId(ctx, mux.Vars(r)["claim_id"])})
	}).Methods("GET")

	router.HandleFunc("/legible/evidence_mgmt/v1/audit/{audit_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		audit, found := app.EvidenceMgmtKeeper.GetAudit(ctx, mux.Vars(r)["audit_id"])
		if !found {
			writeError(w, http.StatusNotFound, "audit not found")
			return
		}
		writeJSON(w, map[string]interface{}{"audit": audit})
	}).Methods("GET")

	router.HandleFunc("/legible/evidence_mgmt/v1/audits/{evidence_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"audits": app.EvidenceMgmtKeeper.GetAuditsForEvidence(ctx, mux.Vars(r)["evidence_id"])})
	}).Methods("GET")

	router.HandleFunc("/legible/evidence_mgmt/v1/oracle/{oracle_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		oracle, found := app.EvidenceMgmtKeeper.GetOracle(ctx, mux.Vars(r)["oracle_id"])
		if !found {
			writeError(w, http.StatusNotFound, "oracle not found")
			return
		}
		writeJSON(w, map[string]interface{}{"oracle": oracle})
	}).Methods("GET")

	router.HandleFunc("/legible/evidence_mgmt/v1/oracles/active", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"oracles": app.EvidenceMgmtKeeper.GetActiveOracles(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/evidence_mgmt/v1/consensus/{evidence_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		c, found := app.EvidenceMgmtKeeper.GetConsensus(ctx, mux.Vars(r)["evidence_id"])
		if !found {
			writeError(w, http.StatusNotFound, "consensus not found")
			return
		}
		writeJSON(w, map[string]interface{}{"consensus": c})
	}).Methods("GET")
}

// --- x/claiming_pot routes (6 endpoints) ---

func registerClaimingPotRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/claiming_pot/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.ClaimingPotKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/claiming_pot/v1/pot/{pot_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		pot, found := app.ClaimingPotKeeper.GetPot(ctx, mux.Vars(r)["pot_id"])
		if !found {
			writeError(w, http.StatusNotFound, "pot not found")
			return
		}
		writeJSON(w, map[string]interface{}{"pot": pot})
	}).Methods("GET")

	router.HandleFunc("/legible/claiming_pot/v1/pots", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		limit := parseUint32(r, "limit", 100)
		if limit > 100 {
			limit = 100
		}
		offset := parseUint32(r, "offset", 0)
		status := r.URL.Query().Get("status")

		var all []*claimingpottypes.ClaimingPot
		if status != "" {
			all = app.ClaimingPotKeeper.GetPotsByStatus(ctx, claimingpottypes.PotStatus(status))
		} else {
			app.ClaimingPotKeeper.IteratePots(ctx, func(pot *claimingpottypes.ClaimingPot) bool {
				all = append(all, pot)
				return false
			})
		}

		total := uint32(len(all))
		start := offset
		if start >= total {
			writeJSON(w, map[string]interface{}{"pots": []claimingpottypes.ClaimingPot{}, "total": total})
			return
		}
		end := start + limit
		if end > total {
			end = total
		}
		writeJSON(w, map[string]interface{}{"pots": all[start:end], "total": total})
	}).Methods("GET")

	router.HandleFunc("/legible/claiming_pot/v1/pots/creator/{creator}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"pots": app.ClaimingPotKeeper.GetPotsByCreator(ctx, mux.Vars(r)["creator"])})
	}).Methods("GET")

	router.HandleFunc("/legible/claiming_pot/v1/eligibility/{pot_id}/{claimer}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		vars := mux.Vars(r)
		pot, found := app.ClaimingPotKeeper.GetPot(ctx, vars["pot_id"])
		if !found {
			writeError(w, http.StatusNotFound, "pot not found")
			return
		}
		eligible, reason := app.ClaimingPotKeeper.EvaluateEligibility(ctx, pot, vars["claimer"])
		writeJSON(w, map[string]interface{}{"eligible": eligible, "reason": reason})
	}).Methods("GET")

	router.HandleFunc("/legible/claiming_pot/v1/records/{pot_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"records": app.ClaimingPotKeeper.GetAllClaimRecords(ctx, mux.Vars(r)["pot_id"])})
	}).Methods("GET")
}

// --- x/channels routes (5 endpoints) ---

func registerChannelsRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/channels/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.ChannelsKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/channels/v1/channel/{channel_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		ch, found := app.ChannelsKeeper.GetChannel(ctx, mux.Vars(r)["channel_id"])
		if !found {
			writeError(w, http.StatusNotFound, "channel not found")
			return
		}
		writeJSON(w, map[string]interface{}{"channel": ch})
	}).Methods("GET")

	router.HandleFunc("/legible/channels/v1/channels/payer/{payer}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"channels": app.ChannelsKeeper.GetChannelsByPayer(ctx, mux.Vars(r)["payer"])})
	}).Methods("GET")

	router.HandleFunc("/legible/channels/v1/channels/provider/{provider}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"channels": app.ChannelsKeeper.GetChannelsByProvider(ctx, mux.Vars(r)["provider"])})
	}).Methods("GET")

	router.HandleFunc("/legible/channels/v1/dispute/{channel_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		d, found := app.ChannelsKeeper.GetDispute(ctx, mux.Vars(r)["channel_id"])
		if !found {
			writeError(w, http.StatusNotFound, "dispute not found")
			return
		}
		writeJSON(w, map[string]interface{}{"dispute": d})
	}).Methods("GET")
}

// --- x/tree routes (7 endpoints) ---

func registerTreeRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/tree/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.TreeKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/tree/v1/project/{project_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		p, found := app.TreeKeeper.GetProject(ctx, mux.Vars(r)["project_id"])
		if !found {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		writeJSON(w, map[string]interface{}{"project": p})
	}).Methods("GET")

	router.HandleFunc("/legible/tree/v1/projects/founder/{founder}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"projects": app.TreeKeeper.GetProjectsByFounder(ctx, mux.Vars(r)["founder"])})
	}).Methods("GET")

	router.HandleFunc("/legible/tree/v1/task/{task_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		t, found := app.TreeKeeper.GetTask(ctx, mux.Vars(r)["task_id"])
		if !found {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeJSON(w, map[string]interface{}{"task": t})
	}).Methods("GET")

	router.HandleFunc("/legible/tree/v1/tasks/project/{project_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"tasks": app.TreeKeeper.GetTasksByProject(ctx, mux.Vars(r)["project_id"])})
	}).Methods("GET")

	router.HandleFunc("/legible/tree/v1/service/{service_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		s, found := app.TreeKeeper.GetService(ctx, mux.Vars(r)["service_id"])
		if !found {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeJSON(w, map[string]interface{}{"service": s})
	}).Methods("GET")

	router.HandleFunc("/legible/tree/v1/seed/{seed_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		s, found := app.TreeKeeper.GetSeed(ctx, mux.Vars(r)["seed_id"])
		if !found {
			writeError(w, http.StatusNotFound, "seed not found")
			return
		}
		writeJSON(w, map[string]interface{}{"seed": s})
	}).Methods("GET")
}

// --- x/bvm routes (6 endpoints) ---

func registerBVMRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/bvm/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.BVMKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/bvm/v1/contract/{address}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		c, found := app.BVMKeeper.GetContract(ctx, mux.Vars(r)["address"])
		if !found {
			writeError(w, http.StatusNotFound, "contract not found")
			return
		}
		writeJSON(w, map[string]interface{}{"contract": c})
	}).Methods("GET")

	router.HandleFunc("/legible/bvm/v1/contracts/creator/{creator}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"contracts": app.BVMKeeper.GetContractsByCreator(ctx, mux.Vars(r)["creator"])})
	}).Methods("GET")

	router.HandleFunc("/legible/bvm/v1/code/{code_hash}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		code, found := app.BVMKeeper.GetCode(ctx, mux.Vars(r)["code_hash"])
		if !found {
			writeError(w, http.StatusNotFound, "code not found")
			return
		}
		writeJSON(w, map[string]interface{}{"code": code})
	}).Methods("GET")

	router.HandleFunc("/legible/bvm/v1/state/{contract_address}/{key}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		vars := mux.Vars(r)
		value, found := app.BVMKeeper.GetContractState(ctx, vars["contract_address"], vars["key"])
		if !found {
			writeError(w, http.StatusNotFound, "state key not found")
			return
		}
		writeJSON(w, map[string]interface{}{"value": value})
	}).Methods("GET")

	router.HandleFunc("/legible/bvm/v1/schedule/{schedule_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		s, found := app.BVMKeeper.GetSchedule(ctx, mux.Vars(r)["schedule_id"])
		if !found {
			writeError(w, http.StatusNotFound, "schedule not found")
			return
		}
		writeJSON(w, map[string]interface{}{"schedule": s})
	}).Methods("GET")
}

// --- x/qualification routes (6 endpoints) ---

func registerQualificationRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/qualification/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.QualificationKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/qualification/v1/qualification/{validator_addr}/{domain}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		vars := mux.Vars(r)
		q, found := app.QualificationKeeper.GetQualification(ctx, vars["validator_addr"], vars["domain"])
		if !found {
			writeError(w, http.StatusNotFound, "qualification not found")
			return
		}
		writeJSON(w, map[string]interface{}{"qualification": q})
	}).Methods("GET")

	router.HandleFunc("/legible/qualification/v1/qualifications/{validator_addr}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"qualifications": app.QualificationKeeper.GetQualificationsByValidator(ctx, mux.Vars(r)["validator_addr"])})
	}).Methods("GET")

	router.HandleFunc("/legible/qualification/v1/qualified/{domain}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"validators": app.QualificationKeeper.GetQualifiedValidators(ctx, mux.Vars(r)["domain"])})
	}).Methods("GET")

	router.HandleFunc("/legible/qualification/v1/endorsement/{endorsement_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		e, found := app.QualificationKeeper.GetEndorsement(ctx, mux.Vars(r)["endorsement_id"])
		if !found {
			writeError(w, http.StatusNotFound, "endorsement not found")
			return
		}
		writeJSON(w, map[string]interface{}{"endorsement": e})
	}).Methods("GET")

	router.HandleFunc("/legible/qualification/v1/endorsements/{validator_addr}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"endorsements": app.QualificationKeeper.GetEndorsementsForValidator(ctx, mux.Vars(r)["validator_addr"])})
	}).Methods("GET")
}

// --- x/capture_challenge routes (3 endpoints) ---

func registerCaptureChallengeRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/capture_challenge/v1/challenge/{challenge_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		ch, found := app.CaptureChallengeKeeper.GetChallenge(ctx, mux.Vars(r)["challenge_id"])
		if !found {
			writeError(w, http.StatusNotFound, "challenge not found")
			return
		}
		writeJSON(w, map[string]interface{}{"challenge": ch})
	}).Methods("GET")

	router.HandleFunc("/legible/capture_challenge/v1/challenges/active", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		domain := r.URL.Query().Get("domain")
		var active []*capturechallengetypes.CaptureChallenge
		app.CaptureChallengeKeeper.IterateChallenges(ctx, func(ch *capturechallengetypes.CaptureChallenge) bool {
			if domain != "" && ch.Domain != domain {
				return false
			}
			switch capturechallengetypes.ChallengeStatus(ch.Status) {
			case capturechallengetypes.ChallengeSubmitted, capturechallengetypes.ChallengeEvidencePeriod, capturechallengetypes.ChallengeUnderReview:
				active = append(active, ch)
			}
			return false
		})
		writeJSON(w, map[string]interface{}{"challenges": active})
	}).Methods("GET")

	router.HandleFunc("/legible/capture_challenge/v1/bounty_pool/{domain}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		pool, found := app.CaptureChallengeKeeper.GetBountyPool(ctx, mux.Vars(r)["domain"])
		if !found {
			writeJSON(w, map[string]interface{}{"pool": nil, "found": false})
			return
		}
		writeJSON(w, map[string]interface{}{"pool": pool, "found": true})
	}).Methods("GET")
}

// --- x/capture_defense routes (4 endpoints) ---

func registerCaptureDefenseRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/capture_defense/v1/qualification/{validator_addr}/{domain}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		vars := mux.Vars(r)
		q, found := app.CaptureDefenseKeeper.GetQualification(ctx, vars["validator_addr"], vars["domain"])
		if !found {
			writeError(w, http.StatusNotFound, "qualification not found")
			return
		}
		writeJSON(w, map[string]interface{}{"qualification": q})
	}).Methods("GET")

	router.HandleFunc("/legible/capture_defense/v1/qualifications/{validator_addr}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"qualifications": app.CaptureDefenseKeeper.GetValidatorQualifications(ctx, mux.Vars(r)["validator_addr"])})
	}).Methods("GET")

	router.HandleFunc("/legible/capture_defense/v1/reputation/{validator_addr}/{domain}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		vars := mux.Vars(r)
		writeJSON(w, map[string]interface{}{"reputation": app.CaptureDefenseKeeper.GetEffectiveReputation(ctx, vars["validator_addr"], vars["domain"])})
	}).Methods("GET")

	router.HandleFunc("/legible/capture_defense/v1/capture_risk/{domain}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		metrics, found := app.CaptureDefenseKeeper.GetCaptureMetrics(ctx, mux.Vars(r)["domain"])
		if !found {
			writeJSON(w, map[string]interface{}{"metrics": nil, "found": false})
			return
		}
		writeJSON(w, map[string]interface{}{"metrics": metrics, "found": true})
	}).Methods("GET")
}

// --- x/ontology routes (10 endpoints) ---

func registerOntologyRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/ontology/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.OntologyKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/ontology/v1/stratum/{stratum}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		n, err2 := strconv.ParseUint(mux.Vars(r)["stratum"], 10, 32)
		if err2 != nil {
			writeError(w, http.StatusBadRequest, "invalid stratum number")
			return
		}
		s := ontologytypes.Stratum(uint32(n))
		props, found := app.OntologyKeeper.GetStratum(ctx, s)
		if !found {
			writeError(w, http.StatusNotFound, "stratum not found")
			return
		}
		writeJSON(w, map[string]interface{}{"properties": props})
	}).Methods("GET")

	router.HandleFunc("/legible/ontology/v1/strata", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"strata": app.OntologyKeeper.GetAllStrata(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/ontology/v1/domain/{name}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		domain, found := app.OntologyKeeper.GetDomain(ctx, mux.Vars(r)["name"])
		if !found {
			writeError(w, http.StatusNotFound, "domain not found")
			return
		}
		writeJSON(w, map[string]interface{}{"domain": domain})
	}).Methods("GET")

	router.HandleFunc("/legible/ontology/v1/domains/stratum/{stratum}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		n, err2 := strconv.ParseUint(mux.Vars(r)["stratum"], 10, 32)
		if err2 != nil {
			writeError(w, http.StatusBadRequest, "invalid stratum number")
			return
		}
		writeJSON(w, map[string]interface{}{"domains": app.OntologyKeeper.GetDomainsByStratum(ctx, ontologytypes.Stratum(uint32(n)))})
	}).Methods("GET")

	router.HandleFunc("/legible/ontology/v1/domains", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"domains": app.OntologyKeeper.GetAllDomains(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/ontology/v1/proposal/{proposal_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		prop, found := app.OntologyKeeper.GetProposal(ctx, mux.Vars(r)["proposal_id"])
		if !found {
			writeError(w, http.StatusNotFound, "proposal not found")
			return
		}
		writeJSON(w, map[string]interface{}{"proposal": prop})
	}).Methods("GET")

	router.HandleFunc("/legible/ontology/v1/confidence_ceiling/{domain_name}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		maxConf, decayRate, err2 := app.OntologyKeeper.GetConfidenceCeiling(ctx, mux.Vars(r)["domain_name"])
		if err2 != nil {
			writeError(w, http.StatusNotFound, err2.Error())
			return
		}
		writeJSON(w, map[string]interface{}{"max_confidence": maxConf, "decay_rate": decayRate, "domain_name": mux.Vars(r)["domain_name"]})
	}).Methods("GET")

	router.HandleFunc("/legible/ontology/v1/logic_zone/{zone}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		props, found := app.OntologyKeeper.GetLogicZone(ctx, ontologytypes.LogicZone(mux.Vars(r)["zone"]))
		if !found {
			writeError(w, http.StatusNotFound, "logic zone not found")
			return
		}
		writeJSON(w, map[string]interface{}{"properties": props})
	}).Methods("GET")

	router.HandleFunc("/legible/ontology/v1/logic_zones", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"zones": app.OntologyKeeper.GetAllLogicZones(ctx)})
	}).Methods("GET")
}

// --- x/research routes (4 endpoints) ---

func registerResearchRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/research/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{
			"params":           app.ResearchKeeper.GetParams(ctx),
			"treasury_balance": app.ResearchKeeper.GetTreasuryBalance(ctx),
		})
	}).Methods("GET")

	router.HandleFunc("/legible/research/v1/research/{research_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		rid := mux.Vars(r)["research_id"]
		research, found := app.ResearchKeeper.GetResearch(ctx, rid)
		if !found {
			writeError(w, http.StatusNotFound, "research not found")
			return
		}
		reviews := app.ResearchKeeper.GetReviewsForResearch(ctx, rid)
		writeJSON(w, map[string]interface{}{"research": research, "reviews": reviews})
	}).Methods("GET")

	router.HandleFunc("/legible/research/v1/submissions", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		status := r.URL.Query().Get("status")
		domain := r.URL.Query().Get("domain")

		var results []*researchtypes.Research
		app.ResearchKeeper.IterateResearches(ctx, func(res *researchtypes.Research) bool {
			if status != "" && string(res.Status) != status {
				return false
			}
			if domain != "" && res.Domain != domain {
				return false
			}
			results = append(results, res)
			return false
		})
		writeJSON(w, map[string]interface{}{"submissions": results})
	}).Methods("GET")

	router.HandleFunc("/legible/research/v1/bounty/{bounty_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		bounty, found := app.ResearchKeeper.GetBounty(ctx, mux.Vars(r)["bounty_id"])
		if !found {
			writeError(w, http.StatusNotFound, "bounty not found")
			return
		}
		writeJSON(w, map[string]interface{}{"bounty": bounty})
	}).Methods("GET")
}

// --- x/schedule routes (3 endpoints) ---

func registerScheduleRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/schedule/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.ScheduleKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/schedule/v1/process/{process_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		p, found := app.ScheduleKeeper.GetProcess(ctx, mux.Vars(r)["process_id"])
		if !found {
			writeError(w, http.StatusNotFound, "process not found")
			return
		}
		writeJSON(w, map[string]interface{}{"process": p})
	}).Methods("GET")

	router.HandleFunc("/legible/schedule/v1/processes/creator/{creator}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"processes": app.ScheduleKeeper.GetProcessesByCreator(ctx, mux.Vars(r)["creator"])})
	}).Methods("GET")
}

// --- x/ibcratelimit routes (2 endpoints) ---

func registerIBCRateLimitRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/ibcratelimit/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.IBCRateLimitKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/ibcratelimit/v1/flow/{channel_id}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		flow, found := app.IBCRateLimitKeeper.GetFlow(ctx, mux.Vars(r)["channel_id"])
		if !found {
			writeJSON(w, map[string]interface{}{"flow": nil, "found": false})
			return
		}
		writeJSON(w, map[string]interface{}{"flow": flow, "found": true})
	}).Methods("GET")
}

// --- x/icaauth routes (2 endpoints) ---

func registerICAAuthRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/icaauth/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.ICAAuthKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/icaauth/v1/accounts/{did}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"accounts": app.ICAAuthKeeper.GetRemoteAccountsForDID(ctx, mux.Vars(r)["did"])})
	}).Methods("GET")
}

func parseUint64(r *http.Request, key string, def uint64) uint64 {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

// --- x/autopoiesis routes (10 endpoints) ---

func registerAutopoiesisRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/autopoiesis/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.AutopoiesisKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/autopoiesis/v1/state", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{
			"state":  app.AutopoiesisKeeper.GetState(ctx),
			"active": app.AutopoiesisKeeper.IsActive(ctx),
		})
	}).Methods("GET")

	router.HandleFunc("/legible/autopoiesis/v1/hormones/{epoch}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		epoch, parseErr := strconv.ParseUint(mux.Vars(r)["epoch"], 10, 64)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid epoch")
			return
		}
		writeJSON(w, map[string]interface{}{"hormones": app.AutopoiesisKeeper.GetHormoneValues(ctx, epoch)})
	}).Methods("GET")

	router.HandleFunc("/legible/autopoiesis/v1/observation/{epoch}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		epoch, parseErr := strconv.ParseUint(mux.Vars(r)["epoch"], 10, 64)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid epoch")
			return
		}
		writeJSON(w, map[string]interface{}{"observation": app.AutopoiesisKeeper.GetObservation(ctx, epoch)})
	}).Methods("GET")

	router.HandleFunc("/legible/autopoiesis/v1/multipliers", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"multipliers": app.AutopoiesisKeeper.GetAllMultiplierEntries(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/autopoiesis/v1/multiplier/{path}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		entry := app.AutopoiesisKeeper.GetMultiplierEntry(ctx, mux.Vars(r)["path"])
		writeJSON(w, map[string]interface{}{"multiplier": entry})
	}).Methods("GET")

	router.HandleFunc("/legible/autopoiesis/v1/adaptations/{epoch}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		epoch, parseErr := strconv.ParseUint(mux.Vars(r)["epoch"], 10, 64)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid epoch")
			return
		}
		writeJSON(w, map[string]interface{}{
			"adaptations": app.AutopoiesisKeeper.GetAdaptations(ctx, epoch),
			"count":       app.AutopoiesisKeeper.GetAdaptationCount(ctx, epoch),
			"conflicts":   app.AutopoiesisKeeper.GetConflictCount(ctx, epoch),
		})
	}).Methods("GET")

	router.HandleFunc("/legible/autopoiesis/v1/learning/state", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"meta_learning": app.AutopoiesisKeeper.GetMetaLearningState(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/autopoiesis/v1/learning/record/{path}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"record": app.AutopoiesisKeeper.GetLearningRecord(ctx, mux.Vars(r)["path"])})
	}).Methods("GET")

	router.HandleFunc("/legible/autopoiesis/v1/proposals/state", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"proposal_state": app.AutopoiesisKeeper.GetAutoProposalState(ctx)})
	}).Methods("GET")
}

// --- x/alignment routes (7 endpoints) ---

func registerAlignmentRoutes(app *LegibleApp, router *mux.Router) {
	router.HandleFunc("/legible/alignment/v1/params", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{"params": app.AlignmentKeeper.GetParams(ctx)})
	}).Methods("GET")

	router.HandleFunc("/legible/alignment/v1/state", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		writeJSON(w, map[string]interface{}{
			"state":  app.AlignmentKeeper.GetState(ctx),
			"active": app.AlignmentKeeper.IsActive(ctx),
		})
	}).Methods("GET")

	router.HandleFunc("/legible/alignment/v1/observation/{epoch}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		epoch, parseErr := strconv.ParseUint(mux.Vars(r)["epoch"], 10, 64)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid epoch")
			return
		}
		writeJSON(w, map[string]interface{}{"observation": app.AlignmentKeeper.GetObservation(ctx, epoch)})
	}).Methods("GET")

	router.HandleFunc("/legible/alignment/v1/scores/{epoch}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		epoch, parseErr := strconv.ParseUint(mux.Vars(r)["epoch"], 10, 64)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid epoch")
			return
		}
		writeJSON(w, map[string]interface{}{"scores": app.AlignmentKeeper.GetScores(ctx, epoch)})
	}).Methods("GET")

	router.HandleFunc("/legible/alignment/v1/corrections/{epoch}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		epoch, parseErr := strconv.ParseUint(mux.Vars(r)["epoch"], 10, 64)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid epoch")
			return
		}
		writeJSON(w, map[string]interface{}{"corrections": app.AlignmentKeeper.GetCorrections(ctx, epoch)})
	}).Methods("GET")

	router.HandleFunc("/legible/alignment/v1/outcomes/{epoch}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		epoch, parseErr := strconv.ParseUint(mux.Vars(r)["epoch"], 10, 64)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid epoch")
			return
		}
		writeJSON(w, map[string]interface{}{"outcomes": app.AlignmentKeeper.GetCorrectionOutcomes(ctx, epoch)})
	}).Methods("GET")

	router.HandleFunc("/legible/alignment/v1/health/{epoch}", func(w http.ResponseWriter, r *http.Request) {
		ctx, err := queryCtx(app)
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "state not available")
			return
		}
		epoch, parseErr := strconv.ParseUint(mux.Vars(r)["epoch"], 10, 64)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid epoch")
			return
		}
		writeJSON(w, map[string]interface{}{"health": app.AlignmentKeeper.GetMetaHealth(ctx, epoch)})
	}).Methods("GET")
}
