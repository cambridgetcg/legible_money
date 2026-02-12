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
	ibc "github.com/cosmos/ibc-go/v8/modules/core"
	ibcporttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibctransfer "github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v8/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"

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

	// gRPC
	"google.golang.org/grpc"

	// gRPC services
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	cmtservice "github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"

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
		discoverymodule.AppModuleBasic{},
		partnershipsmodule.AppModuleBasic{},
		icaauthmodule.AppModuleBasic{},
		ibcratelimitmodule.AppModuleBasic{},
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
		legibleauthtypes.ModuleName:     nil, // holds bootstrap fund, sends to new accounts
		vestingrewardstypes.ModuleName:  {authtypes.Minter, authtypes.Burner},
		knowledgetypes.ModuleName:       nil,
		legiblestakingtypes.ModuleName:  {authtypes.Burner, authtypes.Staking},
		tokenstypes.ModuleName:          {authtypes.Burner},
		ResearchFundName:                {authtypes.Burner},
		BurnModuleName:                  {authtypes.Burner},
		disputestypes.ModuleName:        {authtypes.Burner},
		evidencemgmttypes.ModuleName:    {authtypes.Burner},
		claimingpottypes.ModuleName:     {authtypes.Burner},
		channelstypes.ModuleName:        {authtypes.Burner},
		qualificationtypes.ModuleName:     {authtypes.Burner},
		capturedefensetypes.ModuleName:    {authtypes.Burner},
		capturechallengetypes.ModuleName:  {authtypes.Burner},
		treetypes.ModuleName:              {authtypes.Burner},
		billingtypes.ModuleName:           {authtypes.Burner},
		partnershipstypes.ModuleName:      {authtypes.Burner, authtypes.Staking},
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
	AccountKeeper    authkeeper.AccountKeeper
	BankKeeper       bankkeeper.Keeper
	StakingKeeper    *stakingkeeper.Keeper
	SlashingKeeper   slashingkeeper.Keeper
	DistrKeeper      distrkeeper.Keeper
	GovKeeper        *govkeeper.Keeper
	UpgradeKeeper    *upgradekeeper.Keeper
	EvidenceKeeper   evidencekeeper.Keeper
	FeeGrantKeeper   feegrantkeeper.Keeper
	ConsensusKeeper  consensuskeeper.Keeper

	// ---------- IBC Keepers ----------
	CapabilityKeeper *capabilitykeeper.Keeper
	ScopedIBCKeeper          capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper     capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper
	IBCKeeper            *ibckeeper.Keeper
	IBCFeeKeeper         ibcfeekeeper.Keeper
	TransferKeeper       ibctransferkeeper.Keeper
	ICAControllerKeeper  icacontrollerkeeper.Keeper
	ICAHostKeeper        icahostkeeper.Keeper

	// ---------- Custom Legible Money Keepers ----------
	KnowledgeKeeper      knowledgekeeper.Keeper
	LegibleAuthKeeper    legibleauthkeeper.Keeper
	LegibleStakingKeeper legiblestakingkeeper.Keeper
	EmergencyKeeper      emergencykeeper.Keeper
	TokensKeeper         tokenskeeper.Keeper
	LegibleGovKeeper     lgmgovkeeper.Keeper
	ResearchKeeper       researchkeeper.Keeper
	OntologyKeeper       ontologykeeper.Keeper
	DisputesKeeper       disputeskeeper.Keeper
	EvidenceMgmtKeeper   evidencemgmtkeeper.Keeper
	ClaimingPotKeeper    claimingpotkeeper.Keeper
	ChannelsKeeper       channelskeeper.Keeper
	ScheduleKeeper       schedulekeeper.Keeper
	TreeKeeper           treekeeper.Keeper
	BVMKeeper            bvmkeeper.Keeper
	HomeKeeper           homekeeper.Keeper
	QualificationKeeper    qualificationkeeper.Keeper
	CaptureDefenseKeeper   capturedefensekeeper.Keeper
	CaptureChallengeKeeper capturechallengekeeper.Keeper
	VestingRewardsKeeper   vestingrewardskeeper.Keeper
	BillingKeeper          billingkeeper.Keeper
	DiscoveryKeeper        discoverykeeper.Keeper
	PartnershipsKeeper     partnershipskeeper.Keeper
	ICAAuthKeeper          icaauthkeeper.Keeper
	IBCRateLimitKeeper     ibcratelimitkeeper.Keeper

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
		discoverytypes.StoreKey,
		partnershipstypes.StoreKey,
		icaauthtypes.StoreKey,
		ibcratelimittypes.StoreKey,
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
		rateLimitICS4Wrapper,           // ics4Wrapper — routes through rate limit before IBC core
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
	)

	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		paramstypes.Subspace{}, // x/params removed; IBC transfer accepts empty subspace
		app.IBCFeeKeeper,             // ics4Wrapper — routes through fee → rate limit → IBC core
		app.IBCKeeper.ChannelKeeper,  // channelKeeper
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
		app.TokensKeeper,
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
		nil, // tokensKeeper — nil-safe, wire when needed
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

	app.KnowledgeKeeper = knowledgekeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[knowledgetypes.StoreKey]),
		nil, // memKey
		app.AccountKeeper,
		app.BankKeeper,
		NewStakingKeeperAdapter(app.LegibleStakingKeeper),
		app.TokensKeeper,
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

	// Wire verification hooks: staking (reputation) + qualification (domain metrics)
	app.KnowledgeKeeper.SetVerificationHooks(
		knowledgetypes.NewMultiVerificationHooks(
			legiblestakingkeeper.NewStakingVerificationHooks(app.LegibleStakingKeeper),
			qualificationkeeper.NewQualificationVerificationHooks(app.QualificationKeeper),
		),
	)

	// Wire cross-module keepers into BVM for opcode host functions
	app.BVMKeeper.SetKnowledgeKeeper(NewBVMKnowledgeAdapter(app.KnowledgeKeeper))
	app.BVMKeeper.SetStakingKeeper(NewBVMStakingAdapter(app.LegibleStakingKeeper))
	app.BVMKeeper.SetAuthKeeper(NewBVMAuthAdapter(app.LegibleAuthKeeper))

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
		discoverymodule.NewAppModule(appCodec, app.DiscoveryKeeper),
		partnershipsmodule.NewAppModule(appCodec, app.PartnershipsKeeper),
		icaauthmodule.NewAppModule(appCodec, app.ICAAuthKeeper),
		ibcratelimitmodule.NewAppModule(appCodec, app.IBCRateLimitKeeper),
	)

	// Module init order — standard Cosmos SDK ordering.
	app.ModuleManager.SetOrderBeginBlockers(
		upgradetypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		ibcfeetypes.ModuleName,
		icatypes.ModuleName,
		govtypes.ModuleName,
		feegrant.ModuleName,
		// Custom Legible Money modules
		legibleauthtypes.ModuleName,    // Agent identity (no-op BeginBlock, registered for ordering)
		emergencytypes.ModuleName,      // Emergency ceremony progress (must run before knowledge)
		vestingrewardstypes.ModuleName, // Vesting reward minting (before staking unbondings)
		tokenstypes.ModuleName,         // Sweep expired tokens before unbonding returns
		legiblestakingtypes.ModuleName, // Process mature unbondings
		lgmgovtypes.ModuleName,         // Check voting deadlines
		researchtypes.ModuleName,       // Expire bounties past deadline
		ontologytypes.ModuleName,       // Process expired domain proposals
		disputestypes.ModuleName,           // Process expired disputes
		evidencemgmttypes.ModuleName,       // Expire stale evidence + prune oracles
		claimingpottypes.ModuleName,        // Expire old pots
		channelstypes.ModuleName,           // Resolve expired disputes
		scheduletypes.ModuleName,           // Schedule module (no-op BeginBlock, registered for ordering)
		treetypes.ModuleName,               // Expire seeds, check task deadlines
		bvmtypes.ModuleName,                // Execute pending contract schedules
		hometypes.ModuleName,               // Deadman switches, session cleanup
		qualificationtypes.ModuleName,      // Promote probationary, expire stale qualifications
		capturedefensetypes.ModuleName,     // Expire qualifications, decay reputations
		capturechallengetypes.ModuleName,   // Advance challenge phases
		billingtypes.ModuleName,            // No-op BeginBlock (registered for ordering)
		discoverytypes.ModuleName,          // Expire stale agent profiles
		partnershipstypes.ModuleName,       // Expire formations, lift freezes, expire coercion signals
		icaauthtypes.ModuleName,            // No-op BeginBlock (registered for ordering)
		ibcratelimittypes.ModuleName,       // Reset expired flow records at epoch boundaries
		knowledgetypes.ModuleName,
	)

	app.ModuleManager.SetOrderEndBlockers(
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
		// Custom Legible Money modules
		legibleauthtypes.ModuleName,    // No-op EndBlock (registered for ordering)
		legiblestakingtypes.ModuleName, // Tier transition safety checks
		knowledgetypes.ModuleName,
		tokenstypes.ModuleName,         // Post-block period accounting
		emergencytypes.ModuleName,      // Reset epoch counters
		lgmgovtypes.ModuleName,         // Finalize governance
		researchtypes.ModuleName,
		vestingrewardstypes.ModuleName, // Process vesting schedule releases
		ontologytypes.ModuleName,       // Finalize domain proposals
		disputestypes.ModuleName,           // Finalize disputes
		evidencemgmttypes.ModuleName,       // Evidence end-block cleanup
		claimingpottypes.ModuleName,
		channelstypes.ModuleName,
		scheduletypes.ModuleName,           // Process due schedules + cleanup expired
		treetypes.ModuleName,
		bvmtypes.ModuleName,
		hometypes.ModuleName,               // Cleanup old alerts
		qualificationtypes.ModuleName,
		capturedefensetypes.ModuleName,     // Periodic capture detection
		capturechallengetypes.ModuleName,   // Auto-fund bounty pools
		billingtypes.ModuleName,            // Expire old quotes
		discoverytypes.ModuleName,          // Expire stale agent profiles
		partnershipstypes.ModuleName,       // Settle cooling partnerships
		icaauthtypes.ModuleName,            // No-op EndBlock
		ibcratelimittypes.ModuleName,       // No-op EndBlock
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
		// Custom Legible Money modules (init after standard modules)
		legibleauthtypes.ModuleName,
		legiblestakingtypes.ModuleName,
		vestingrewardstypes.ModuleName,
		tokenstypes.ModuleName,
		emergencytypes.ModuleName,
		lgmgovtypes.ModuleName,
		researchtypes.ModuleName,
		ontologytypes.ModuleName,
		disputestypes.ModuleName,
		evidencemgmttypes.ModuleName,
		claimingpottypes.ModuleName,
		channelstypes.ModuleName,
		scheduletypes.ModuleName,
		treetypes.ModuleName,
		bvmtypes.ModuleName,
		hometypes.ModuleName,
		qualificationtypes.ModuleName,
		capturedefensetypes.ModuleName,
		capturechallengetypes.ModuleName,
		billingtypes.ModuleName,
		discoverytypes.ModuleName,
		partnershipstypes.ModuleName,
		icaauthtypes.ModuleName,
		ibcratelimittypes.ModuleName,
		knowledgetypes.ModuleName,
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
	app.VoteExtConfig = &VoteExtensionConfig{
		LocalStore: NewLocalCommitmentStore(),
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
	for _, tx := range req.Txs {
		if isVoteExtInjectionTx(tx) {
			app.ProcessVoteExtInjection(ctx, tx)
			break // at most one injection tx per block
		}
	}
	return &sdk.ResponsePreBlock{}, nil
}

// BeginBlocker runs begin-block logic for all modules.
func (app *LegibleApp) BeginBlocker(ctx sdk.Context) (sdk.BeginBlock, error) {
	// Route accumulated fees: 22% to research fund, 78% stays in fee_collector
	// for standard Cosmos distribution to validators.
	if err := RouteFees(ctx, app.BankKeeper); err != nil {
		ctx.Logger().Error("fee routing failed", "err", err)
	}

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
type SafeServiceRegistrar struct {
	inner  grpc.ServiceRegistrar
	logger log.Logger
}

func (s *SafeServiceRegistrar) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Warn("skipped gRPC service registration (proto types not generated)",
				"service", desc.ServiceName, "reason", fmt.Sprintf("%v", r))
		}
	}()
	s.inner.RegisterService(desc, impl)
}
