// Package app implements the Legible Money Cosmos SDK application.
//
// Legible Money is a blockchain for AI agent economies using Proof of Truth (PoT)
// consensus, where verifying knowledge IS the useful work.
//
// This file registers all standard Cosmos SDK modules plus custom Legible Money
// modules. The MVC (Minimum Viable Cosmos) phase starts with standard modules
// for IBC connectivity, then custom modules are added incrementally.
package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cast"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/evidence"
	evidencekeeper "cosmossdk.io/x/evidence/keeper"
	evidencetypes "cosmossdk.io/x/evidence/types"
	"cosmossdk.io/x/feegrant"
	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	feegrantmodule "cosmossdk.io/x/feegrant/module"
	"cosmossdk.io/x/tx/signing"
	"cosmossdk.io/x/upgrade"
	upgradekeeper "cosmossdk.io/x/upgrade/keeper"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/auth/vesting"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	consensuskeeper "github.com/cosmos/cosmos-sdk/x/consensus/keeper"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	// IBC modules
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	ibcporttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	// ICA (Interchain Accounts) modules
	ica "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts"
	icacontroller "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"

	// IBC Fee Middleware (ICS-29)
	ibcfee "github.com/cosmos/ibc-go/v8/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v8/modules/apps/29-fee/types"

	// CometBFT
	abci "github.com/cometbft/cometbft/abci/types"

	// Crypto codec for keyring support
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"

	// Tx types (cosmos.tx.v1beta1.Tx registration)
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"

	// gRPC
	"google.golang.org/grpc"

	// gRPC services
	cmtservice "github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"

	// Custom Legible Money modules
	knowledgemodule "github.com/legible-money/legible/x/knowledge"
	knowledgekeeper "github.com/legible-money/legible/x/knowledge/keeper"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"

	legibleauth "github.com/legible-money/legible/x/auth"
	legibleauthkeeper "github.com/legible-money/legible/x/auth/keeper"
	legibleauthtypes "github.com/legible-money/legible/x/auth/types"

	legiblestaking "github.com/legible-money/legible/x/staking"
	legiblestakingkeeper "github.com/legible-money/legible/x/staking/keeper"
	legiblestakingtypes "github.com/legible-money/legible/x/staking/types"

	emergencymodule "github.com/legible-money/legible/x/emergency"
	emergencykeeper "github.com/legible-money/legible/x/emergency/keeper"
	emergencytypes "github.com/legible-money/legible/x/emergency/types"

	tokensmodule "github.com/legible-money/legible/x/tokens"
	tokenskeeper "github.com/legible-money/legible/x/tokens/keeper"
	tokenstypes "github.com/legible-money/legible/x/tokens/types"

	lgmgov "github.com/legible-money/legible/x/gov"
	lgmgovkeeper "github.com/legible-money/legible/x/gov/keeper"
	lgmgovtypes "github.com/legible-money/legible/x/gov/types"

	researchmodule "github.com/legible-money/legible/x/research"
	researchkeeper "github.com/legible-money/legible/x/research/keeper"
	researchtypes "github.com/legible-money/legible/x/research/types"

	vestingrewardsmodule "github.com/legible-money/legible/x/vesting_rewards"
	vestingrewardskeeper "github.com/legible-money/legible/x/vesting_rewards/keeper"
	vestingrewardstypes "github.com/legible-money/legible/x/vesting_rewards/types"

	ontologymodule "github.com/legible-money/legible/x/ontology"
	ontologykeeper "github.com/legible-money/legible/x/ontology/keeper"
	ontologytypes "github.com/legible-money/legible/x/ontology/types"

	disputesmodule "github.com/legible-money/legible/x/disputes"
	disputeskeeper "github.com/legible-money/legible/x/disputes/keeper"
	disputestypes "github.com/legible-money/legible/x/disputes/types"

	evidencemgmtmodule "github.com/legible-money/legible/x/evidence_mgmt"
	evidencemgmtkeeper "github.com/legible-money/legible/x/evidence_mgmt/keeper"
	evidencemgmttypes "github.com/legible-money/legible/x/evidence_mgmt/types"

	claimingpotmodule "github.com/legible-money/legible/x/claiming_pot"
	claimingpotkeeper "github.com/legible-money/legible/x/claiming_pot/keeper"
	claimingpottypes "github.com/legible-money/legible/x/claiming_pot/types"

	channelsmodule "github.com/legible-money/legible/x/channels"
	channelskeeper "github.com/legible-money/legible/x/channels/keeper"
	channelstypes "github.com/legible-money/legible/x/channels/types"

	schedulemodule "github.com/legible-money/legible/x/schedule"
	schedulekeeper "github.com/legible-money/legible/x/schedule/keeper"
	scheduletypes "github.com/legible-money/legible/x/schedule/types"

	treemodule "github.com/legible-money/legible/x/tree"
	treekeeper "github.com/legible-money/legible/x/tree/keeper"
	treetypes "github.com/legible-money/legible/x/tree/types"

	bvmmodule "github.com/legible-money/legible/x/bvm"
	bvmkeeper "github.com/legible-money/legible/x/bvm/keeper"
	bvmtypes "github.com/legible-money/legible/x/bvm/types"

	homemodule "github.com/legible-money/legible/x/home"
	homekeeper "github.com/legible-money/legible/x/home/keeper"
	hometypes "github.com/legible-money/legible/x/home/types"

	qualificationmodule "github.com/legible-money/legible/x/qualification"
	qualificationkeeper "github.com/legible-money/legible/x/qualification/keeper"
	qualificationtypes "github.com/legible-money/legible/x/qualification/types"

	capturedefensemodule "github.com/legible-money/legible/x/capture_defense"
	capturedefensekeeper "github.com/legible-money/legible/x/capture_defense/keeper"
	capturedefensetypes "github.com/legible-money/legible/x/capture_defense/types"

	capturechallengemodule "github.com/legible-money/legible/x/capture_challenge"
	capturechallengekeeper "github.com/legible-money/legible/x/capture_challenge/keeper"
	capturechallengetypes "github.com/legible-money/legible/x/capture_challenge/types"

	billingmodule "github.com/legible-money/legible/x/billing"
	billingkeeper "github.com/legible-money/legible/x/billing/keeper"
	billingtypes "github.com/legible-money/legible/x/billing/types"

	autopoiesismodule "github.com/legible-money/legible/x/autopoiesis"
	autopoiesiskeeper "github.com/legible-money/legible/x/autopoiesis/keeper"
	autopoiesistypes "github.com/legible-money/legible/x/autopoiesis/types"

	discoverymodule "github.com/legible-money/legible/x/discovery"
	discoverykeeper "github.com/legible-money/legible/x/discovery/keeper"
	discoverytypes "github.com/legible-money/legible/x/discovery/types"

	partnershipsmodule "github.com/legible-money/legible/x/partnerships"
	partnershipskeeper "github.com/legible-money/legible/x/partnerships/keeper"
	partnershipstypes "github.com/legible-money/legible/x/partnerships/types"

	icaauthmodule "github.com/legible-money/legible/x/icaauth"
	icaauthkeeper "github.com/legible-money/legible/x/icaauth/keeper"
	icaauthtypes "github.com/legible-money/legible/x/icaauth/types"

	ibcratelimitmodule "github.com/legible-money/legible/x/ibcratelimit"
	ibcratelimitkeeper "github.com/legible-money/legible/x/ibcratelimit/keeper"
	ibcratelimittypes "github.com/legible-money/legible/x/ibcratelimit/types"

	alignmentmodule "github.com/legible-money/legible/x/alignment"
	alignmentkeeper "github.com/legible-money/legible/x/alignment/keeper"
	alignmenttypes "github.com/legible-money/legible/x/alignment/types"

	computepoolmodule "github.com/legible-money/legible/x/compute_pool"
	computepoolkeeper "github.com/legible-money/legible/x/compute_pool/keeper"
	computepooltypes "github.com/legible-money/legible/x/compute_pool/types"
)

const (
	// AppName is the application name.
	AppName = "legbled"

	// AccountAddressPrefix is the bech32 prefix for LGM addresses.
	AccountAddressPrefix = "lgm"

	// BondDenom is the staking denomination.
	BondDenom = "ulgm"

	// DisplayDenom is the human-readable denomination.
	DisplayDenom = "lgm"

	// DefaultBlockTime is the target block time in milliseconds.
	DefaultBlockTime = 2521
)

var (
	// DefaultNodeHome is the default home directory for the node.
	DefaultNodeHome string

	// ModuleBasics defines the module BasicManager used for codec registration
	// and genesis verification.
	ModuleBasics = module.NewBasicManager(
		auth.AppModuleBasic{},
		genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(nil),
		slashing.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		vesting.AppModuleBasic{},
		consensus.AppModuleBasic{},
		ibc.AppModuleBasic{},
		ibctransfer.AppModuleBasic{},
		ibcfee.AppModuleBasic{},
		ica.AppModuleBasic{},
		// Custom Legible Money modules
		knowledgemodule.AppModuleBasic{},
		legibleauth.AppModuleBasic{},
		legiblestaking.AppModuleBasic{},
		emergencymodule.AppModuleBasic{},
		tokensmodule.AppModuleBasic{},
		lgmgov.AppModuleBasic{},
		researchmodule.AppModuleBasic{},
		vestingrewardsmodule.AppModuleBasic{},
		ontologymodule.AppModuleBasic{},
		disputesmodule.AppModuleBasic{},
		evidencemgmtmodule.AppModuleBasic{},
		claimingpotmodule.AppModuleBasic{},
		channelsmodule.AppModuleBasic{},
		schedulemodule.AppModuleBasic{},
		treemodule.AppModuleBasic{},
		bvmmodule.AppModuleBasic{},
		homemodule.AppModuleBasic{},
		qualificationmodule.AppModuleBasic{},
		capturedefensemodule.AppModuleBasic{},
		capturechallengemodule.AppModuleBasic{},
		billingmodule.AppModuleBasic{},
		autopoiesismodule.AppModuleBasic{},
		discoverymodule.AppModuleBasic{},
		partnershipsmodule.AppModuleBasic{},
		icaauthmodule.AppModuleBasic{},
		ibcratelimitmodule.AppModuleBasic{},
		alignmentmodule.AppModuleBasic{},
		computepoolmodule.AppModuleBasic{},
	)

	// Module account permissions.
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		ibcfeetypes.ModuleName:         nil,
		icatypes.ModuleName:            nil,
		// Custom Legible Money module accounts
		legibleauthtypes.ModuleName:      nil, // holds bootstrap fund, sends to new accounts
		vestingrewardstypes.ModuleName:   {authtypes.Minter, authtypes.Burner},
		knowledgetypes.ModuleName:        nil,
		legiblestakingtypes.ModuleName:   {authtypes.Burner, authtypes.Staking},
		tokenstypes.ModuleName:           {authtypes.Burner},
		ResearchFundName:                 nil, // receives 7% research tax; no mint/burn needed
		BurnModuleName:                   {authtypes.Burner},
		disputestypes.ModuleName:         {authtypes.Burner},
		evidencemgmttypes.ModuleName:     {authtypes.Burner},
		claimingpottypes.ModuleName:      {authtypes.Burner},
		channelstypes.ModuleName:         {authtypes.Burner},
		qualificationtypes.ModuleName:    {authtypes.Burner},
		capturedefensetypes.ModuleName:   {authtypes.Burner},
		capturechallengetypes.ModuleName: {authtypes.Burner},
		treetypes.ModuleName:             {authtypes.Burner},
		billingtypes.ModuleName:          {authtypes.Burner},
		partnershipstypes.ModuleName:     {authtypes.Burner, authtypes.Staking},
		computepooltypes.ModuleName:      {authtypes.Burner},
	}
)

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	DefaultNodeHome = filepath.Join(userHomeDir, ".legbled")

	// Set bech32 prefixes for LGM addresses.
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(AccountAddressPrefix, AccountAddressPrefix+"pub")
	config.SetBech32PrefixForValidator(AccountAddressPrefix+"valoper", AccountAddressPrefix+"valoperpub")
	config.SetBech32PrefixForConsensusNode(AccountAddressPrefix+"valcons", AccountAddressPrefix+"valconspub")
	config.Seal()

	// Set the default bond denom to ulgm (micro-LGM).
	sdk.DefaultBondDenom = "ulgm"
}

// LegibleApp extends baseapp.BaseApp with all Cosmos SDK modules.
type LegibleApp struct {
	*baseapp.BaseApp

	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	txConfig          client.TxConfig
	interfaceRegistry types.InterfaceRegistry

	// Store keys
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// ---------- Standard Cosmos SDK Keepers ----------
	AccountKeeper   authkeeper.AccountKeeper
	BankKeeper      bankkeeper.Keeper
	StakingKeeper   *stakingkeeper.Keeper
	SlashingKeeper  slashingkeeper.Keeper
	DistrKeeper     distrkeeper.Keeper
	GovKeeper       *govkeeper.Keeper
	UpgradeKeeper   *upgradekeeper.Keeper
	EvidenceKeeper  evidencekeeper.Keeper
	FeeGrantKeeper  feegrantkeeper.Keeper
	ConsensusKeeper consensuskeeper.Keeper

	// ---------- IBC Keepers ----------
	CapabilityKeeper          *capabilitykeeper.Keeper
	ScopedIBCKeeper           capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper      capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper
	IBCKeeper                 *ibckeeper.Keeper
	IBCFeeKeeper              ibcfeekeeper.Keeper
	TransferKeeper            ibctransferkeeper.Keeper
	ICAControllerKeeper       icacontrollerkeeper.Keeper
	ICAHostKeeper             icahostkeeper.Keeper

	// ---------- Custom Legible Money Keepers ----------
	KnowledgeKeeper        knowledgekeeper.Keeper
	LegibleAuthKeeper      legibleauthkeeper.Keeper
	LegibleStakingKeeper   legiblestakingkeeper.Keeper
	EmergencyKeeper        emergencykeeper.Keeper
	TokensKeeper           tokenskeeper.Keeper
	LegibleGovKeeper       lgmgovkeeper.Keeper
	ResearchKeeper         researchkeeper.Keeper
	OntologyKeeper         ontologykeeper.Keeper
	DisputesKeeper         disputeskeeper.Keeper
	EvidenceMgmtKeeper     evidencemgmtkeeper.Keeper
	ClaimingPotKeeper      claimingpotkeeper.Keeper
	ChannelsKeeper         channelskeeper.Keeper
	ScheduleKeeper         schedulekeeper.Keeper
	TreeKeeper             treekeeper.Keeper
	BVMKeeper              bvmkeeper.Keeper
	HomeKeeper             homekeeper.Keeper
	QualificationKeeper    qualificationkeeper.Keeper
	CaptureDefenseKeeper   capturedefensekeeper.Keeper
	CaptureChallengeKeeper capturechallengekeeper.Keeper
	VestingRewardsKeeper   vestingrewardskeeper.Keeper
	BillingKeeper          billingkeeper.Keeper
	AutopoiesisKeeper      autopoiesiskeeper.Keeper
	DiscoveryKeeper        discoverykeeper.Keeper
	PartnershipsKeeper     partnershipskeeper.Keeper
	ICAAuthKeeper          icaauthkeeper.Keeper
	IBCRateLimitKeeper     ibcratelimitkeeper.Keeper
	AlignmentKeeper        alignmentkeeper.Keeper
	ComputePoolKeeper      computepoolkeeper.Keeper

	// Vote extension config for PoT consensus (node-local, set at startup)
	VoteExtConfig *VoteExtensionConfig

	// Module manager
	ModuleManager *module.Manager

	// Simulation manager (for fuzz testing)
	sm *module.SimulationManager

	// Configurator for module msg/query registration
	configurator module.Configurator
}

// NewLegibleApp creates and initializes a new LegibleApp instance.
func NewLegibleApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	appOpts servertypes.AppOptions,
	baseAppOptions ...func(*baseapp.BaseApp),
) *LegibleApp {
	interfaceRegistry, err := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          addresscodec.NewBech32Codec(AccountAddressPrefix),
			ValidatorAddressCodec: addresscodec.NewBech32Codec(AccountAddressPrefix + "valoper"),
		},
	})
	if err != nil {
		panic(err)
	}
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()
	txConfig := authtx.NewTxConfig(appCodec, authtx.DefaultSignModes)

	// Register standard SDK and module interfaces so that RegisterServices can
	// find message types (e.g. evidence, staking, gov).
	sdk.RegisterLegacyAminoCodec(legacyAmino)
	sdk.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	ModuleBasics.RegisterLegacyAminoCodec(legacyAmino)
	txtypes.RegisterInterfaces(interfaceRegistry)

	bApp := baseapp.NewBaseApp(AppName, logger, db, txConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	// ---- Store Keys ----
	keys := storetypes.NewKVStoreKeys(
		authtypes.StoreKey,
		banktypes.StoreKey,
		stakingtypes.StoreKey,
		distrtypes.StoreKey,
		slashingtypes.StoreKey,
		govtypes.StoreKey,
		upgradetypes.StoreKey,
		feegrant.StoreKey,
		evidencetypes.StoreKey,
		capabilitytypes.StoreKey,
		ibcexported.StoreKey,
		ibctransfertypes.StoreKey,
		ibcfeetypes.StoreKey,
		icacontrollertypes.StoreKey,
		icahosttypes.StoreKey,
		"consensus", // x/consensus module store key
		// Custom Legible Money module store keys
		knowledgetypes.StoreKey,
		legibleauthtypes.StoreKey,
		legiblestakingtypes.StoreKey,
		emergencytypes.StoreKey,
		tokenstypes.StoreKey,
		lgmgovtypes.StoreKey,
		researchtypes.StoreKey,
		vestingrewardstypes.StoreKey,
		ontologytypes.StoreKey,
		disputestypes.StoreKey,
		evidencemgmttypes.StoreKey,
		claimingpottypes.StoreKey,
		channelstypes.StoreKey,
		scheduletypes.StoreKey,
		treetypes.StoreKey,
		bvmtypes.StoreKey,
		hometypes.StoreKey,
		qualificationtypes.StoreKey,
		capturedefensetypes.StoreKey,
		capturechallengetypes.StoreKey,
		billingtypes.StoreKey,
		autopoiesistypes.StoreKey,
		discoverytypes.StoreKey,
		partnershipstypes.StoreKey,
		icaauthtypes.StoreKey,
		ibcratelimittypes.StoreKey,
		alignmenttypes.StoreKey,
		computepooltypes.StoreKey,
	)
	tkeys := storetypes.NewTransientStoreKeys()
	memKeys := storetypes.NewMemoryStoreKeys(capabilitytypes.MemStoreKey)

	app := &LegibleApp{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		txConfig:          txConfig,
		interfaceRegistry: interfaceRegistry,
		keys:              keys,
		tkeys:             tkeys,
		memKeys:           memKeys,
	}

	// ---- Module Keepers ----

	app.ConsensusKeeper = consensuskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys["consensus"]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		runtime.EventService{},
	)
	bApp.SetParamStore(app.ConsensusKeeper.ParamsStore)

	app.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[authtypes.StoreKey]),
		authtypes.ProtoBaseAccount,
		maccPerms,
		addresscodec.NewBech32Codec(AccountAddressPrefix),
		AccountAddressPrefix,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[banktypes.StoreKey]),
		app.AccountKeeper,
		blockedModuleAccountAddrs(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		logger,
	)

	app.StakingKeeper = stakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[stakingtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		addresscodec.NewBech32Codec(AccountAddressPrefix+"valoper"),
		addresscodec.NewBech32Codec(AccountAddressPrefix+"valcons"),
	)

	app.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[distrtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		legacyAmino,
		runtime.NewKVStoreService(keys[slashingtypes.StoreKey]),
		app.StakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[feegrant.StoreKey]),
		app.AccountKeeper,
	)

	app.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights(appOpts),
		runtime.NewKVStoreService(keys[upgradetypes.StoreKey]),
		appCodec,
		DefaultNodeHome,
		app.BaseApp,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.EvidenceKeeper = *evidencekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[evidencetypes.StoreKey]),
		app.StakingKeeper,
		app.SlashingKeeper,
		app.AccountKeeper.AddressCodec(),
		runtime.ProvideCometInfoService(),
	)

	// ---- Staking Hooks ----
	// Wire the slashing and distribution modules as hooks on the staking module
	// so that validator signing info is created when validators are added during genesis.
	app.StakingKeeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(app.DistrKeeper.Hooks(), app.SlashingKeeper.Hooks()),
	)

	// ---- Capability Keeper (required by IBC) ----

	app.CapabilityKeeper = capabilitykeeper.NewKeeper(
		appCodec,
		keys[capabilitytypes.StoreKey],
		memKeys[capabilitytypes.MemStoreKey],
	)
	app.ScopedIBCKeeper = app.CapabilityKeeper.ScopeToModule(ibcexported.ModuleName)
	app.ScopedTransferKeeper = app.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	app.ScopedICAControllerKeeper = app.CapabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)
	app.ScopedICAHostKeeper = app.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)

	// Seal the capability keeper after all ScopeToModule() calls.
	// This prevents any module from claiming new capabilities at runtime,
	// which would be a capability escalation attack vector.
	app.CapabilityKeeper.Seal()

	// ---- IBC Keepers ----

	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		keys[ibcexported.StoreKey],
		paramstypes.Subspace{}, // x/params removed; IBC accepts empty subspace (v0.47+ self-manages params)
		app.StakingKeeper,
		app.UpgradeKeeper,
		app.ScopedIBCKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// ---- IBC Rate Limit Keeper ----
	// Must be created before IBCFeeKeeper so it can serve as the outermost ics4Wrapper.
	// Outgoing packet chain: TransferKeeper → IBCFeeKeeper → RateLimitICS4Wrapper → ChannelKeeper
	app.IBCRateLimitKeeper = ibcratelimitkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ibcratelimittypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Rate limit ICS4Wrapper wraps the raw ChannelKeeper to check outflow limits before sending.
	rateLimitICS4Wrapper := ibcratelimitkeeper.NewICS4Wrapper(app.IBCKeeper.ChannelKeeper, app.IBCRateLimitKeeper)

	// ---- IBC Fee Middleware Keeper (ICS-29) ----
	// The fee keeper's ics4Wrapper is the rate limit wrapper, so outgoing packets
	// flow through: fee → rate limit → channel keeper.
	app.IBCFeeKeeper = ibcfeekeeper.NewKeeper(
		appCodec,
		keys[ibcfeetypes.StoreKey],
		rateLimitICS4Wrapper, // ics4Wrapper — routes through rate limit before IBC core
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
	)

	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		paramstypes.Subspace{},      // x/params removed; IBC transfer accepts empty subspace
		app.IBCFeeKeeper,            // ics4Wrapper — routes through fee → rate limit → IBC core
		app.IBCKeeper.ChannelKeeper, // channelKeeper
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		app.ScopedTransferKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// ---- ICA Keepers ----

	app.ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
		appCodec,
		keys[icacontrollertypes.StoreKey],
		paramstypes.Subspace{},
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.ScopedICAControllerKeeper,
		app.MsgServiceRouter(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec,
		keys[icahosttypes.StoreKey],
		paramstypes.Subspace{},
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.ScopedICAHostKeeper,
		app.MsgServiceRouter(),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.ICAHostKeeper.WithQueryRouter(app.GRPCQueryRouter())

	// ---- Custom Legible Money Keepers ----

	app.LegibleAuthKeeper = legibleauthkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[legibleauthtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.TokensKeeper = tokenskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[tokenstypes.StoreKey]),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.LegibleStakingKeeper = legiblestakingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[legiblestakingtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.LegibleAuthKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.EmergencyKeeper = emergencykeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[emergencytypes.StoreKey]),
		app.LegibleStakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.LegibleGovKeeper = lgmgovkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[lgmgovtypes.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		NewStakeQueryAdapter(app.LegibleStakingKeeper),
		app.BankKeeper,
		NewLGMGovUpgradeAdapter(app.UpgradeKeeper),
	)

	app.ResearchKeeper = researchkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[researchtypes.StoreKey]),
		app.BankKeeper,
		NewStakeQueryAdapter(app.LegibleStakingKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.VestingRewardsKeeper = vestingrewardskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[vestingrewardstypes.StoreKey]),
		app.BankKeeper,
		NewVestingStakingAdapter(app.LegibleStakingKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.OntologyKeeper = ontologykeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[ontologytypes.StoreKey]),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.EvidenceMgmtKeeper = evidencemgmtkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[evidencemgmttypes.StoreKey]),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.ClaimingPotKeeper = claimingpotkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[claimingpottypes.StoreKey]),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.ChannelsKeeper = channelskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[channelstypes.StoreKey]),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.ScheduleKeeper = schedulekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[scheduletypes.StoreKey]),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.TreeKeeper = treekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[treetypes.StoreKey]),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.TreeKeeper.SetChannelsKeeper(app.ChannelsKeeper)

	app.BVMKeeper = bvmkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[bvmtypes.StoreKey]),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.QualificationKeeper = qualificationkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[qualificationtypes.StoreKey]),
		app.OntologyKeeper,
		app.LegibleStakingKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.CaptureDefenseKeeper = capturedefensekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[capturedefensetypes.StoreKey]),
		NewStakingKeeperAdapter(app.LegibleStakingKeeper),
		NewCaptureDefenseOntologyAdapter(app.OntologyKeeper),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.CaptureChallengeKeeper = capturechallengekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[capturechallengetypes.StoreKey]),
		app.CaptureDefenseKeeper,
		NewStakingKeeperAdapter(app.LegibleStakingKeeper),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.HomeKeeper = homekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[hometypes.StoreKey]),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.AutopoiesisKeeper = autopoiesiskeeper.NewKeeper(
		runtime.NewKVStoreService(keys[autopoiesistypes.StoreKey]),
		logger,
		nil, // stakingKeeper — set after staking keeper init (nil-safe)
		nil, // knowledgeKeeper — set after knowledge keeper init (nil-safe)
		nil, // emergencyKeeper — set after emergency keeper init (nil-safe)
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.DiscoveryKeeper = discoverykeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[discoverytypes.StoreKey]),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.PartnershipsKeeper = partnershipskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[partnershipstypes.StoreKey]),
		app.BankKeeper,
		NewPartnershipsStakingAdapter(app.LegibleStakingKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.ICAAuthKeeper = icaauthkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[icaauthtypes.StoreKey]),
		app.LegibleAuthKeeper,
		NewICAControllerAdapter(app.ICAControllerKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	knowledgeStakingAdapter := NewStakingKeeperAdapter(app.LegibleStakingKeeper)
	knowledgeStakingAdapter.SetSDKStakingKeeper(app.StakingKeeper)

	app.KnowledgeKeeper = knowledgekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[knowledgetypes.StoreKey]),
		nil, // memKey
		app.AccountKeeper,
		app.BankKeeper,
		knowledgeStakingAdapter,
		app.QualificationKeeper,
		app.CaptureDefenseKeeper,
		app.VestingRewardsKeeper,
		NewEvidenceMgmtKeeperAdapter(app.EvidenceMgmtKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Wire cross-module knowledge→evidence_mgmt reference (init ordering)
	app.EvidenceMgmtKeeper.SetKnowledgeKeeper(&KnowledgeKeeperForEvidence{keeper: app.KnowledgeKeeper})

	// Wire billing keeper (depends on KnowledgeKeeper for fact pricing)
	app.BillingKeeper = billingkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[billingtypes.StoreKey]),
		app.BankKeeper,
		NewBillingKnowledgeAdapter(app.KnowledgeKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// Wire ontology keeper into knowledge for dynamic stratum ceilings
	app.KnowledgeKeeper.SetOntologyKeeper(app.OntologyKeeper)

	// Wire home keeper into knowledge for epistemic activity notifications
	// (verifications, claims, challenges → agent home patina scores)
	app.KnowledgeKeeper.SetHomeKeeper(app.HomeKeeper)

	// Wire verification hooks: staking (reputation) + qualification (domain metrics)
	app.KnowledgeKeeper.SetVerificationHooks(
		knowledgetypes.NewMultiVerificationHooks(
			legiblestakingkeeper.NewStakingVerificationHooks(app.LegibleStakingKeeper),
			qualificationkeeper.NewQualificationVerificationHooks(app.QualificationKeeper),
		),
	)

	// Wire knowledge keeper into autopoiesis for openclaw_critical trigger
	app.AutopoiesisKeeper.SetKnowledgeKeeper(NewAutopoiesisKnowledgeAdapter(app.KnowledgeKeeper))

	// Wire cross-module keepers into BVM for opcode host functions
	app.BVMKeeper.SetKnowledgeKeeper(NewBVMKnowledgeAdapter(app.KnowledgeKeeper))
	app.BVMKeeper.SetStakingKeeper(NewBVMStakingAdapter(app.LegibleStakingKeeper))
	app.BVMKeeper.SetAuthKeeper(NewBVMAuthAdapter(app.LegibleAuthKeeper))

	// ---- Alignment Keeper (philosophical alignment feedback loop) ----
	app.AlignmentKeeper = alignmentkeeper.NewKeeper(
		runtime.NewKVStoreService(keys[alignmenttypes.StoreKey]),
		logger,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	// Cross-module wiring for alignment sensors (nil-safe: stubs return defaults)
	app.AlignmentKeeper.SetAutopoiesisKeeper(NewAlignmentAutopoiesisAdapter(app.AutopoiesisKeeper))
	app.AlignmentKeeper.SetEmergencyKeeper(NewAlignmentEmergencyAdapter(app.EmergencyKeeper))
	app.AlignmentKeeper.SetKnowledgeKeeper(NewAlignmentKnowledgeAdapter(app.KnowledgeKeeper))
	app.AlignmentKeeper.SetStakingKeeper(NewAlignmentStakingAdapter(app.LegibleStakingKeeper))
	app.AlignmentKeeper.SetOntologyKeeper(NewAlignmentOntologyAdapter(app.OntologyKeeper))
	app.AlignmentKeeper.SetVestingKeeper(NewAlignmentVestingRewardsAdapter(app.VestingRewardsKeeper))

	// Wire autopoiesis → staking for SSI slash multiplier
	app.LegibleStakingKeeper.SetAutopoiesisKeeper(app.AutopoiesisKeeper)

	// Wire governance → autopoiesis for auto-proposal rate limiting (Loop 4)
	app.AutopoiesisKeeper.SetGovKeeper(NewAutopoiesisGovAdapter(app.LegibleGovKeeper))

	app.DisputesKeeper = disputeskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[disputestypes.StoreKey]),
		app.BankKeeper,
		NewDisputeKnowledgeAdapter(app.KnowledgeKeeper),
		NewDisputeStakingAdapter(app.LegibleStakingKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.DisputesKeeper.SetVestingKeeper(NewDisputeVestingAdapter(app.VestingRewardsKeeper))

	// Wire emergency keeper into capture defense for halt-aware analysis
	app.CaptureDefenseKeeper.SetEmergencyKeeper(app.EmergencyKeeper)

	// ---- Gov Keeper (Cosmos SDK standard governance) ----
	//
	// DUAL GOVERNANCE: Legible Money runs TWO governance systems:
	//
	// 1. Cosmos SDK x/gov ("gov") — Standard text/parameter/upgrade proposals
	//    with deposit + voting periods. Used for SDK-level parameter changes
	//    and standard governance proposals. Store: "gov".
	//
	// 2. LGM x/gov ("lgm_gov") — Legible Improvement Proposals (LIPs) with
	//    quadratic voting and category-based stake requirements. Handles
	//    protocol-specific governance (consensus, economics, interface, process).
	//    Store: "lgm_gov". Bridges to x/upgrade for software upgrades.
	//
	// The SDK x/gov is required for IBC parameter governance and standard
	// Cosmos ecosystem compatibility. The LGM x/gov extends it with
	// protocol-specific governance features (quadratic voting, LIP lifecycle,
	// treasury disbursements, and software upgrade scheduling).

	govConfig := govtypes.DefaultConfig()
	govKeeper := govkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[govtypes.StoreKey]),
		app.AccountKeeper,
		app.BankKeeper,
		app.StakingKeeper,
		app.DistrKeeper,
		bApp.MsgServiceRouter(),
		govConfig,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.GovKeeper = govKeeper

	govRouter := govv1beta1.NewRouter()
	govRouter.
		AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler)

	govKeeper.SetLegacyRouter(govRouter)

	// ---- IBC Router ----

	ibcRouter := ibcporttypes.NewRouter()

	// IBC middleware stack for transfer:
	// Inbound:  IBC core → rate limit → fee → transfer module
	// Outbound: transfer → fee keeper → rate limit ICS4Wrapper → channel keeper
	transferIBCModule := ibctransfer.NewIBCModule(app.TransferKeeper)
	feeTransferModule := ibcfee.NewIBCMiddleware(transferIBCModule, app.IBCFeeKeeper)
	rateLimitTransferModule := ibcratelimitkeeper.NewIBCMiddleware(feeTransferModule, app.IBCRateLimitKeeper)
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, rateLimitTransferModule)

	// ICA routes
	icaControllerIBCModule := icacontroller.NewIBCMiddleware(nil, app.ICAControllerKeeper)
	icaHostIBCModule := icahost.NewIBCModule(app.ICAHostKeeper)
	ibcRouter.AddRoute(icacontrollertypes.SubModuleName, icaControllerIBCModule)
	ibcRouter.AddRoute(icahosttypes.SubModuleName, icaHostIBCModule)

	// Custom IBC modules (Phase 2: knowledge oracle IBC module)
	// ibcRouter.AddRoute("knowledge", knowledgeIBCModule)
	app.IBCKeeper.SetRouter(ibcRouter)

	// ---- Module Manager ----

	app.ModuleManager = module.NewManager(
		genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app, app.txConfig),
		auth.NewAppModule(appCodec, app.AccountKeeper, nil, paramstypes.Subspace{}),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, paramstypes.Subspace{}),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper, paramstypes.Subspace{}),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, paramstypes.Subspace{}, app.interfaceRegistry),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, paramstypes.Subspace{}),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, paramstypes.Subspace{}),
		upgrade.NewAppModule(app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		evidence.NewAppModule(app.EvidenceKeeper),
		consensus.NewAppModule(appCodec, app.ConsensusKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		ibctransfer.NewAppModule(app.TransferKeeper),
		ibcfee.NewAppModule(app.IBCFeeKeeper),
		ica.NewAppModule(&app.ICAControllerKeeper, &app.ICAHostKeeper),
		// Custom Legible Money modules
		knowledgemodule.NewAppModule(appCodec, app.KnowledgeKeeper),
		legibleauth.NewAppModule(appCodec, app.LegibleAuthKeeper),
		legiblestaking.NewAppModule(appCodec, app.LegibleStakingKeeper),
		emergencymodule.NewAppModule(appCodec, app.EmergencyKeeper),
		tokensmodule.NewAppModule(appCodec, app.TokensKeeper),
		lgmgov.NewAppModule(appCodec, app.LegibleGovKeeper),
		researchmodule.NewAppModule(appCodec, app.ResearchKeeper),
		vestingrewardsmodule.NewAppModule(appCodec, app.VestingRewardsKeeper),
		ontologymodule.NewAppModule(appCodec, app.OntologyKeeper),
		disputesmodule.NewAppModule(appCodec, app.DisputesKeeper),
		evidencemgmtmodule.NewAppModule(appCodec, app.EvidenceMgmtKeeper),
		claimingpotmodule.NewAppModule(appCodec, app.ClaimingPotKeeper),
		channelsmodule.NewAppModule(appCodec, app.ChannelsKeeper),
		schedulemodule.NewAppModule(appCodec, app.ScheduleKeeper),
		treemodule.NewAppModule(appCodec, app.TreeKeeper),
		bvmmodule.NewAppModule(appCodec, app.BVMKeeper),
		homemodule.NewAppModule(appCodec, app.HomeKeeper),
		qualificationmodule.NewAppModule(appCodec, app.QualificationKeeper),
		capturedefensemodule.NewAppModule(appCodec, app.CaptureDefenseKeeper),
		capturechallengemodule.NewAppModule(appCodec, app.CaptureChallengeKeeper),
		billingmodule.NewAppModule(appCodec, app.BillingKeeper),
		autopoiesismodule.NewAppModule(appCodec, app.AutopoiesisKeeper),
		discoverymodule.NewAppModule(appCodec, app.DiscoveryKeeper),
		partnershipsmodule.NewAppModule(appCodec, app.PartnershipsKeeper),
		icaauthmodule.NewAppModule(appCodec, app.ICAAuthKeeper),
		ibcratelimitmodule.NewAppModule(appCodec, app.IBCRateLimitKeeper),
		alignmentmodule.NewAppModule(appCodec, app.AlignmentKeeper),
	)

	// =======================================================================
	// BeginBlocker Ordering
	//
	// CRITICAL: Do not reorder without verifying all dependency chains.
	// Each module's position is justified by its dependencies on state
	// that earlier modules produce. Mis-ordering can cause silent state
	// corruption or consensus failures.
	//
	// Dependency chains (A → B means A must run before B):
	//   upgrade → ALL            (halts chain at upgrade height before any module touches state)
	//   SDK staking → SDK slashing (jailing depends on validator set)
	//   auth → staking           (session cleanup before stake operations)
	//   emergency → capture_defense (halt status must be known for reputation decay)
	//   vesting_rewards → staking (minted rewards before unbonding credits)
	//   tokens → staking         (period sweep before OriginPeriod credits)
	//   staking → knowledge      (validator set must be current for round phases)
	//   ontology → qualification (domain structure before qualification checks)
	//   qualification → knowledge (validator eligibility before round selection)
	//   knowledge → LAST         (depends on most other modules' state being current)
	// =======================================================================
	app.ModuleManager.SetOrderBeginBlockers(
		// ── SDK infrastructure (must run first) ──
		upgradetypes.ModuleName,        // MUST be first: halts chain at scheduled upgrade height
		vestingrewardstypes.ModuleName, // MUST run before x/distribution: RouteFees() intercepts
		// 7% research tax from fee_collector before distribution
		// sweeps remaining 93% to validators. Also mints block rewards.
		distrtypes.ModuleName,       // Distribution rewards calculation (after fee routing)
		slashingtypes.ModuleName,    // Validator jailing/unjailing
		evidencetypes.ModuleName,    // SDK evidence processing
		stakingtypes.ModuleName,     // SDK validator set updates
		ibcexported.ModuleName,      // IBC client updates
		ibctransfertypes.ModuleName, // IBC transfer timeouts
		ibcfeetypes.ModuleName,      // IBC fee processing
		icatypes.ModuleName,         // Interchain accounts
		govtypes.ModuleName,         // SDK governance
		feegrant.ModuleName,         // Fee grant expiration

		// ── LGM Tier 1: Identity & safety (no cross-module deps) ──
		legibleauthtypes.ModuleName, // Session cleanup, recovery timeouts. Runs early so auth
		// state is current before staking/knowledge use it.
		emergencytypes.ModuleName, // Ceremony progress. MUST run before capture_defense
		// so halt status is known for reputation decay.

		// ── LGM Tier 2: Economic foundation ──
		tokenstypes.ModuleName, // Sweep expired periods, RAI distribution. MUST run
		// before staking so period data is current for OriginPeriod.
		legiblestakingtypes.ModuleName, // Process mature unbondings, credit with OriginPeriod.
		// MUST run before knowledge so validator set is current.

		// ── LGM Tier 3: Governance & research (independent of each other) ──
		lgmgovtypes.ModuleName,   // Check voting deadlines, advance expired proposals.
		researchtypes.ModuleName, // Expire bounties past deadline.

		// ── LGM Tier 4: Domain & evidence (independent of each other) ──
		ontologytypes.ModuleName, // Process domain proposals. MUST run before qualification
		// so domain structure is current for qualification checks.
		disputestypes.ModuleName,     // Process expired disputes.
		evidencemgmttypes.ModuleName, // Expire stale evidence, prune oracles.
		claimingpottypes.ModuleName,  // Expire old claiming pots.
		channelstypes.ModuleName,     // Resolve expired channel disputes.

		// ── LGM Tier 5: Execution & agent infrastructure ──
		scheduletypes.ModuleName, // No-op BeginBlock (processes in EndBlock).
		treetypes.ModuleName,     // Expire seeds, check task deadlines.
		bvmtypes.ModuleName,      // Execute pending contract schedules.
		hometypes.ModuleName,     // Deadman switches, session cleanup, treasury, aging.

		// ── LGM Tier 6: Qualification & safety (depends on Tier 4) ──
		qualificationtypes.ModuleName, // Promote/expire qualifications. MUST run after ontology
		// so domain structure is current.
		capturedefensetypes.ModuleName, // Reputation decay, qualification expiry. MUST run
		// after emergency (halt awareness).
		capturechallengetypes.ModuleName, // Advance challenge phases.

		// ── LGM Tier 7: Leaf modules (no dependents) ──
		autopoiesistypes.ModuleName,  // Autopoiesis: no-op BeginBlock (observe in EndBlock).
		alignmenttypes.ModuleName,    // Alignment: no-op BeginBlock (sense in EndBlock).
		billingtypes.ModuleName,      // No-op BeginBlock.
		discoverytypes.ModuleName,    // Expire stale agent profiles.
		partnershipstypes.ModuleName, // Expire formations, lift freezes.
		icaauthtypes.ModuleName,      // No-op BeginBlock.
		ibcratelimittypes.ModuleName, // Reset expired flow records at epoch boundaries.

		// ── LGM LAST: Knowledge (central hub, depends on most modules) ──
		knowledgetypes.ModuleName, // Adversarial BeginBlocker, round phase transitions.
		// Runs LAST because it depends on: staking (validator set),
		// tokens (period data), qualification (eligibility),
		// capture_defense (reputation), emergency (halt status).
	)

	// Machine-checked: panics at startup if declared dependency edges are violated.
	ValidateBlockerOrder(app.ModuleManager.OrderBeginBlockers, beginBlockerDeps)

	// =======================================================================
	// EndBlocker Ordering
	//
	// EndBlockers finalize state for the block. Less ordering-sensitive than
	// BeginBlockers because they process completions rather than preparations.
	// Key constraint: knowledge should run early so fact expirations are
	// visible to later modules (disputes, evidence) in the same block.
	// =======================================================================
	app.ModuleManager.SetOrderEndBlockers(
		// ── SDK infrastructure ──
		upgradetypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		ibcfeetypes.ModuleName,
		icatypes.ModuleName,
		feegrant.ModuleName,

		// ── LGM: Core state finalization ──
		legibleauthtypes.ModuleName,    // No-op EndBlock.
		legiblestakingtypes.ModuleName, // Tier transition safety checks.
		knowledgetypes.ModuleName,      // Expire stale facts. Runs early so expired facts
		// are visible to disputes/evidence in same block.
		tokenstypes.ModuleName,    // Post-block period accounting.
		emergencytypes.ModuleName, // Reset per-epoch proposal counts.

		// ── LGM: Governance & economics ──
		lgmgovtypes.ModuleName,         // Finalize governance tallies.
		researchtypes.ModuleName,       // No-op EndBlock.
		vestingrewardstypes.ModuleName, // Process vesting schedule releases.

		// ── LGM: Domain & evidence ──
		ontologytypes.ModuleName,     // Compute dynamic stratum ceilings.
		disputestypes.ModuleName,     // Finalize dispute resolutions.
		evidencemgmttypes.ModuleName, // Reliability scoring, evidence cleanup.
		claimingpottypes.ModuleName,  // No-op EndBlock.
		channelstypes.ModuleName,     // No-op EndBlock.

		// ── LGM: Execution & agent ──
		scheduletypes.ModuleName, // Process due schedules, cleanup expired.
		treetypes.ModuleName,     // No-op EndBlock.
		bvmtypes.ModuleName,      // No-op EndBlock.
		hometypes.ModuleName,     // Cleanup old acknowledged alerts.

		// ── LGM: Safety ──
		qualificationtypes.ModuleName,    // No-op EndBlock.
		capturedefensetypes.ModuleName,   // Periodic capture detection analysis.
		capturechallengetypes.ModuleName, // Auto-fund bounty pools for high-risk domains.

		// ── LGM: Leaf modules ──
		autopoiesistypes.ModuleName,  // Autopoiesis: OBSERVE → COMPUTE → ADAPT at epoch boundaries.
		alignmenttypes.ModuleName,    // Alignment: SENSE → JUDGE → CORRECT → REFLECT. MUST run
		// AFTER autopoiesis so it can override multiplier paths at priority 5.
		billingtypes.ModuleName,      // Expire old API quotes.
		discoverytypes.ModuleName,    // Expire stale agent profiles.
		partnershipstypes.ModuleName, // Settle cooling partnerships.
		icaauthtypes.ModuleName,      // No-op EndBlock.
		ibcratelimittypes.ModuleName, // No-op EndBlock.
	)

	app.ModuleManager.SetOrderInitGenesis(
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		ibcexported.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		ibctransfertypes.ModuleName,
		ibcfeetypes.ModuleName,
		icatypes.ModuleName,
		feegrant.ModuleName,
		upgradetypes.ModuleName,
		vestingtypes.ModuleName,
		// ── Custom Legible Money modules ──
		// InitGenesis order mirrors keeper construction order in app.go.
		// auth → staking → tokens must init first (other modules depend on them).
		// knowledge inits LAST because it queries staking/qualification/ontology.
		legibleauthtypes.ModuleName,      // Auth: accounts, DIDs, session keys.
		legiblestakingtypes.ModuleName,   // Staking: validators, tiers (needs auth).
		vestingrewardstypes.ModuleName,   // Vesting: reward schedules (needs staking).
		tokenstypes.ModuleName,           // Tokens: period ledger (needs bank).
		emergencytypes.ModuleName,        // Emergency: genesis council, params.
		lgmgovtypes.ModuleName,           // Governance: LIPs, disbursements.
		researchtypes.ModuleName,         // Research: bounties.
		ontologytypes.ModuleName,         // Ontology: domains, strata (before qualification).
		disputestypes.ModuleName,         // Disputes: params.
		evidencemgmttypes.ModuleName,     // Evidence: params.
		claimingpottypes.ModuleName,      // Claiming pot: params.
		channelstypes.ModuleName,         // Channels: params.
		scheduletypes.ModuleName,         // Schedule: params.
		treetypes.ModuleName,             // Tree: seeds.
		bvmtypes.ModuleName,              // BVM: deployed contracts.
		hometypes.ModuleName,             // Home: agent homes.
		qualificationtypes.ModuleName,    // Qualification: domain qualifications (needs ontology).
		capturedefensetypes.ModuleName,   // Capture defense: reputation scores.
		capturechallengetypes.ModuleName, // Capture challenge: active challenges.
		autopoiesistypes.ModuleName,      // Autopoiesis: multipliers, SSI state.
		alignmenttypes.ModuleName,        // Alignment: philosophical feedback loop params.
		billingtypes.ModuleName,          // Billing: providers, quotes.
		discoverytypes.ModuleName,        // Discovery: agent profiles.
		partnershipstypes.ModuleName,     // Partnerships: formation records.
		icaauthtypes.ModuleName,          // ICA auth: ICA entries.
		ibcratelimittypes.ModuleName,     // IBC rate limit: flow records.
		knowledgetypes.ModuleName,        // Knowledge: facts, claims, domains. LAST because
		// it depends on staking, ontology, qualification.
	)

	// Use a safe configurator that recovers from panics in RegisterService for
	// modules without proto-generated type URLs (hand-written gRPC descriptors).
	// This allows the chain to boot even when proto codegen hasn't been run for
	// custom modules — they simply won't have gRPC endpoints until codegen is done.
	safeMsgRouter := &SafeServiceRegistrar{inner: bApp.MsgServiceRouter(), logger: logger}
	safeQueryRouter := &SafeServiceRegistrar{inner: bApp.GRPCQueryRouter(), logger: logger}
	app.configurator = module.NewConfigurator(appCodec, safeMsgRouter, safeQueryRouter)
	if err = app.ModuleManager.RegisterServices(app.configurator); err != nil {
		panic(err)
	}

	// Register upgrade handlers (must be after RegisterServices, before LoadLatestVersion)
	app.registerUpgradeHandlers()

	// Mount stores
	app.MountKVStores(keys)
	app.MountTransientStores(tkeys)
	app.MountMemoryStores(memKeys)

	// Set AnteHandler
	app.SetAnteHandler(NewAnteHandler(app))

	// Set InitChainer
	app.SetInitChainer(app.InitChainer)
	app.SetPreBlocker(app.PreBlocker)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)

	// ---- ABCI++ Handlers for Proof of Truth Consensus ----

	// PrepareProposal and ProcessProposal handle block building + validation.
	app.SetPrepareProposal(app.PrepareProposalHandler())
	app.SetProcessProposal(app.ProcessProposalHandler())

	// Vote extension handlers attach PoT verification data to validator votes.
	// ValidatorAddress and ValidatorPrivateKey are set by the node operator at
	// startup (via environment or config). Until configured, ExtendVote returns
	// empty extensions (non-validator nodes or unconfigured validators).
	// Resolve data directory for PoT commitment persistence.
	// Commitments survive node restarts so validators don't forfeit reveals.
	homeDir := cast.ToString(appOpts.Get("home"))
	if homeDir == "" {
		homeDir = DefaultNodeHome
	}
	potDataDir := filepath.Join(homeDir, "data")
	app.VoteExtConfig = &VoteExtensionConfig{
		LocalStore: NewLocalCommitmentStore(potDataDir),
	}
	app.SetExtendVoteHandler(ExtendVoteHandler(app.KnowledgeKeeper, app.VoteExtConfig))
	app.SetVerifyVoteExtensionHandler(VerifyVoteExtensionHandler(app.KnowledgeKeeper))

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			logger.Error("error loading latest version", "err", err)
			os.Exit(1)
		}
	}

	return app
}

// InitChainer initializes the chain from genesis.
func (app *LegibleApp) InitChainer(ctx sdk.Context, req *abci.RequestInitChain) (*abci.ResponseInitChain, error) {
	var genesisState map[string]json.RawMessage
	if err := json.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}
	resp, err := app.ModuleManager.InitGenesis(ctx, app.appCodec, genesisState)
	if err != nil {
		return nil, err
	}

	// Workaround: cosmos-db GoLevelDB returns nil for keys with empty values,
	// but IAVL's SaveEmptyRoot writes []byte{} as value. This causes GetRoot
	// to return ErrVersionDoesNotExist for any IAVL store that has no keys
	// written during genesis, breaking all gRPC/REST state queries. Ensure
	// every KV store has at least one key so no IAVL tree is empty at commit.
	app.ensureNonEmptyStores(ctx)

	return resp, nil
}

// ensureNonEmptyStores writes a sentinel key to any KV store that would
// otherwise be empty after genesis. This prevents the GoLevelDB empty-value
// bug in IAVL from making stores unqueryable.
//
// Root cause: cosmos/cosmos-db GoLevelDB adapter returns nil from Get() for
// keys with empty-byte values ([]byte{}), but IAVL SaveEmptyRoot writes
// []byte{} as the root. IAVL's GetRoot nil-check then reports the version
// as nonexistent. This affects ANY Cosmos SDK chain using GoLevelDB with
// modules that write no keys during InitGenesis.
//
// REMOVE THIS when cosmos/cosmos-db GoLevelDB Get() is patched to return
// []byte{} (not nil) for empty-value keys, or when switching to PebbleDB/
// RocksDB which do not exhibit this behavior.
func (app *LegibleApp) ensureNonEmptyStores(ctx sdk.Context) {
	sentinel := []byte("__store_initialized")
	for _, key := range app.keys {
		store := ctx.KVStore(key)
		if !store.Has(sentinel) {
			store.Set(sentinel, []byte{0x01})
		}
	}
}

// PreBlocker runs before BeginBlocker on each block. It processes the vote
// extension injection pseudo-tx (if present) so that commitments and reveals
// are stored in keeper state before the knowledge module's BeginBlocker
// advances round phases.
func (app *LegibleApp) PreBlocker(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
	// Count real transactions (excluding vote extension injection pseudo-txs)
	realTxCount := 0
	for _, tx := range req.Txs {
		if isVoteExtInjectionTx(tx) {
			app.ProcessVoteExtInjection(ctx, tx)
		} else {
			realTxCount++
		}
	}
	// Store tx count for vesting_rewards BeginBlocker empty-block detection.
	// CometBFT DataHash is always >0 bytes even for empty blocks, so we pass
	// the actual count from the FinalizeBlock request.
	app.VestingRewardsKeeper.SetBlockTxCount(realTxCount)

	return &sdk.ResponsePreBlock{}, nil
}

// BeginBlocker runs begin-block logic for all modules.
// Fee routing (7% research / 93% validators) is handled in x/vesting_rewards BeginBlocker.
func (app *LegibleApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	return app.ModuleManager.BeginBlock(ctx)
}

// EndBlocker runs end-block logic for all modules.
func (app *LegibleApp) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
	return app.ModuleManager.EndBlock(ctx)
}

// RegisterAPIRoutes registers REST API routes.
func (app *LegibleApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	authtypes.RegisterInterfaces(clientCtx.InterfaceRegistry)
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register custom REST routes for hand-written module types.
	registerCustomRESTRoutes(app, apiSvr.Router)
}

// LegacyAmino returns the legacy amino codec.
func (app *LegibleApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns the protobuf codec.
func (app *LegibleApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns the interface registry.
func (app *LegibleApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// DefaultGenesis returns the default genesis state for the application.
func (app *LegibleApp) DefaultGenesis() map[string]json.RawMessage {
	return ModuleBasics.DefaultGenesis(app.appCodec)
}

// SimulationManager returns the simulation manager.
func (app *LegibleApp) SimulationManager() *module.SimulationManager {
	return app.sm
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *LegibleApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *LegibleApp) RegisterTendermintService(clientCtx client.Context) {
	cmtservice.RegisterTendermintService(
		clientCtx,
		app.BaseApp.GRPCQueryRouter(),
		app.interfaceRegistry,
		app.Query,
	)
}

// RegisterNodeService implements the Application.RegisterNodeService method.
func (app *LegibleApp) RegisterNodeService(clientCtx client.Context, cfg config.Config) {
	nodeservice.RegisterNodeService(clientCtx, app.GRPCQueryRouter(), cfg)
}

// LoadHeight loads the application state at a given height.
func (app *LegibleApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ExportAppStateAndValidators exports the application state for genesis and returns validators.
func (app *LegibleApp) ExportAppStateAndValidators(
	forZeroHeight bool,
	jailAllowedAddrs []string,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {
	ctx := app.NewContext(true)

	// Optionally reset heights to zero for re-genesis.
	if forZeroHeight {
		app.prepForZeroHeightGenesis(ctx, jailAllowedAddrs)
	}

	genState, err := app.ModuleManager.ExportGenesisForModules(ctx, app.appCodec, modulesToExport)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	appState, err := json.Marshal(genState)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	validators, err := staking.WriteValidators(ctx, app.StakingKeeper)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	return servertypes.ExportedApp{
		AppState:        appState,
		Validators:      validators,
		Height:          app.LastBlockHeight(),
		ConsensusParams: app.GetConsensusParams(ctx),
	}, nil
}

// prepForZeroHeightGenesis prepares the application state for a fresh genesis
// at height zero. This is used when exporting state for a chain restart.
// It withdraws all outstanding rewards and commissions, resets slashing info,
// and re-initializes distribution state so the exported genesis is self-consistent.
func (app *LegibleApp) prepForZeroHeightGenesis(ctx sdk.Context, jailAllowedAddrs []string) {
	// Build allow-list: these validators will NOT be jailed on re-genesis
	allowedAddrsMap := make(map[string]bool)
	for _, addr := range jailAllowedAddrs {
		allowedAddrsMap[addr] = true
	}

	// 1. Withdraw all validator commissions
	app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			return false
		}
		_, _ = app.DistrKeeper.WithdrawValidatorCommission(ctx, valBz)
		return false
	})

	// 2. Withdraw all delegator rewards
	dels, err := app.StakingKeeper.GetAllDelegations(ctx)
	if err != nil {
		ctx.Logger().Error("prepForZeroHeightGenesis: failed to get delegations", "err", err)
		return
	}
	for _, del := range dels {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(del.GetValidatorAddr())
		if err != nil {
			continue
		}
		delBz := del.GetDelegatorAddr()
		delAddr, err := app.AccountKeeper.AddressCodec().StringToBytes(delBz)
		if err != nil {
			continue
		}
		_, _ = app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, valBz)
	}

	// 3. Clear all validator slash events and historical rewards
	app.DistrKeeper.DeleteAllValidatorSlashEvents(ctx)
	app.DistrKeeper.DeleteAllValidatorHistoricalRewards(ctx)

	// 4. Set block height to 0 for re-initialization
	ctx = ctx.WithBlockHeight(0)

	// 5. Reinitialize all validators in distribution
	app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			return false
		}
		// Sweep any outstanding rewards into the community pool
		scraps, err := app.DistrKeeper.GetValidatorOutstandingRewardsCoins(ctx, valBz)
		if err == nil && !scraps.IsZero() {
			feePool, fpErr := app.DistrKeeper.FeePool.Get(ctx)
			if fpErr == nil {
				feePool.CommunityPool = feePool.CommunityPool.Add(scraps...)
				_ = app.DistrKeeper.FeePool.Set(ctx, feePool)
			}
		}
		return false
	})

	// 6. Reinitialize all delegations in distribution
	for _, del := range dels {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(del.GetValidatorAddr())
		if err != nil {
			continue
		}
		delBz := del.GetDelegatorAddr()
		delAddr, err := app.AccountKeeper.AddressCodec().StringToBytes(delBz)
		if err != nil {
			continue
		}
		_ = app.DistrKeeper.Hooks().BeforeDelegationCreated(ctx, delAddr, valBz)
		_ = app.DistrKeeper.Hooks().AfterDelegationModified(ctx, delAddr, valBz)
	}

	// 7. Reset slashing signing info start heights to 0
	app.SlashingKeeper.IterateValidatorSigningInfos(ctx, func(addr sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) (stop bool) {
		info.StartHeight = 0
		app.SlashingKeeper.SetValidatorSigningInfo(ctx, addr, info)
		return false
	})
}

// ---------- Helpers ----------

func blockedModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}
	return modAccAddrs
}

func skipUpgradeHeights(appOpts servertypes.AppOptions) map[int64]bool {
	skipHeights := make(map[int64]bool)
	for _, h := range cast.ToIntSlice(appOpts.Get("unsafe-skip-upgrades")) {
		skipHeights[int64(h)] = true
	}
	return skipHeights
}

// EncodingConfig holds the codec, interface registry, and tx config used by
// the CLI and app initialization. Exported so root.go can create a client
// context with a real codec before the app is instantiated.
type EncodingConfig struct {
	InterfaceRegistry types.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

// MakeEncodingConfig creates the encoding config for the CLI layer.
// This mirrors the codec setup inside NewLegibleApp so that commands like
// `init`, `validate-genesis`, and `gentx` can marshal/unmarshal genesis
// state before the full app is constructed.
func MakeEncodingConfig() EncodingConfig {
	interfaceRegistry, err := types.NewInterfaceRegistryWithOptions(types.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          addresscodec.NewBech32Codec(AccountAddressPrefix),
			ValidatorAddressCodec: addresscodec.NewBech32Codec(AccountAddressPrefix + "valoper"),
		},
	})
	if err != nil {
		panic(err)
	}
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	legacyAmino := codec.NewLegacyAmino()
	txConfig := authtx.NewTxConfig(appCodec, authtx.DefaultSignModes)

	// Register standard SDK types (crypto keys, tx types) needed by the keyring
	// and genesis commands before the full app is constructed.
	sdk.RegisterLegacyAminoCodec(legacyAmino)
	sdk.RegisterInterfaces(interfaceRegistry)
	cryptocodec.RegisterInterfaces(interfaceRegistry)

	// Register module interfaces so the codec can handle
	// genesis types (e.g. slashing, staking, gov).
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	ModuleBasics.RegisterLegacyAminoCodec(legacyAmino)
	txtypes.RegisterInterfaces(interfaceRegistry)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             appCodec,
		TxConfig:          txConfig,
		Amino:             legacyAmino,
	}
}

// SafeServiceRegistrar wraps a gRPC ServiceRegistrar and recovers from panics
// caused by unregistered proto type URLs. This allows modules with hand-written
// (non-proto-generated) message types to skip gRPC registration gracefully
// instead of crashing the node at startup.
//
// TODO: Remove once all modules have proto-generated .pb.go files. Track
// progress by checking FailedServices after RegisterServices completes.
type SafeServiceRegistrar struct {
	inner          grpc.ServiceRegistrar
	logger         log.Logger
	FailedServices []string // Services that failed registration (for diagnostics)
}

func (s *SafeServiceRegistrar) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	defer func() {
		if r := recover(); r != nil {
			s.FailedServices = append(s.FailedServices, desc.ServiceName)
			s.logger.Error("gRPC service registration FAILED — module has no gRPC endpoints",
				"service", desc.ServiceName, "reason", fmt.Sprintf("%v", r))
		}
	}()
	s.inner.RegisterService(desc, impl)
}
