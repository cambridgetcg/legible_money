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
	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
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

	// CometBFT
	abci "github.com/cometbft/cometbft/abci/types"

	// Crypto codec for keyring support
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"

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
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.NewAppModuleBasic(
			[]govclient.ProposalHandler{
				paramsclient.ProposalHandler,
			},
		),
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
		feegrantmodule.AppModuleBasic{},
		upgrade.AppModuleBasic{},
		evidence.AppModuleBasic{},
		vesting.AppModuleBasic{},
		consensus.AppModuleBasic{},
		ibc.AppModuleBasic{},
		ibctransfer.AppModuleBasic{},
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
		treemodule.AppModuleBasic{},
		bvmmodule.AppModuleBasic{},
		homemodule.AppModuleBasic{},
		qualificationmodule.AppModuleBasic{},
		capturedefensemodule.AppModuleBasic{},
		capturechallengemodule.AppModuleBasic{},
	)

	// Module account permissions.
	maccPerms = map[string][]string{
		authtypes.FeeCollectorName:     nil,
		distrtypes.ModuleName:          nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		govtypes.ModuleName:            {authtypes.Burner},
		ibctransfertypes.ModuleName:    {authtypes.Minter, authtypes.Burner},
		// Custom Legible Money module accounts
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
	MintKeeper       mintkeeper.Keeper
	DistrKeeper      distrkeeper.Keeper
	GovKeeper        *govkeeper.Keeper
	CrisisKeeper     *crisiskeeper.Keeper
	UpgradeKeeper    *upgradekeeper.Keeper
	ParamsKeeper     paramskeeper.Keeper
	EvidenceKeeper   evidencekeeper.Keeper
	FeeGrantKeeper   feegrantkeeper.Keeper
	ConsensusKeeper  consensuskeeper.Keeper

	// ---------- IBC Keepers ----------
	CapabilityKeeper *capabilitykeeper.Keeper
	ScopedIBCKeeper  capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper capabilitykeeper.ScopedKeeper
	IBCKeeper        *ibckeeper.Keeper
	TransferKeeper   ibctransferkeeper.Keeper

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
	TreeKeeper           treekeeper.Keeper
	BVMKeeper            bvmkeeper.Keeper
	HomeKeeper           homekeeper.Keeper
	QualificationKeeper    qualificationkeeper.Keeper
	CaptureDefenseKeeper   capturedefensekeeper.Keeper
	CaptureChallengeKeeper capturechallengekeeper.Keeper
	VestingRewardsKeeper   vestingrewardskeeper.Keeper

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
		crisistypes.StoreKey,
		minttypes.StoreKey,
		distrtypes.StoreKey,
		slashingtypes.StoreKey,
		govtypes.StoreKey,
		paramstypes.StoreKey,
		upgradetypes.StoreKey,
		feegrant.StoreKey,
		evidencetypes.StoreKey,
		capabilitytypes.StoreKey,
		ibcexported.StoreKey,
		ibctransfertypes.StoreKey,
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
		treetypes.StoreKey,
		bvmtypes.StoreKey,
		hometypes.StoreKey,
		qualificationtypes.StoreKey,
		capturedefensetypes.StoreKey,
		capturechallengetypes.StoreKey,
	)
	tkeys := storetypes.NewTransientStoreKeys(paramstypes.TStoreKey)
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

	app.ParamsKeeper = initParamsKeeper(appCodec, legacyAmino, keys[paramstypes.StoreKey], tkeys[paramstypes.TStoreKey])

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

	app.MintKeeper = mintkeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[minttypes.StoreKey]),
		app.StakingKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
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

	app.CrisisKeeper = crisiskeeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[crisistypes.StoreKey]),
		cast.ToUint(appOpts.Get("inv-check-period")),
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		app.AccountKeeper.AddressCodec(),
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

	// ---- IBC Keepers ----

	app.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		keys[ibcexported.StoreKey],
		app.GetSubspace(ibcexported.ModuleName),
		app.StakingKeeper,
		app.UpgradeKeeper,
		app.ScopedIBCKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		keys[ibctransfertypes.StoreKey],
		app.GetSubspace(ibctransfertypes.ModuleName),
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.ChannelKeeper,
		app.IBCKeeper.PortKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		app.ScopedTransferKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	// ---- Custom Legible Money Keepers ----

	app.LegibleAuthKeeper = legibleauthkeeper.NewKeeper(
		appCodec,
		keys[legibleauthtypes.StoreKey],
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.TokensKeeper = tokenskeeper.NewKeeper(
		appCodec,
		keys[tokenstypes.StoreKey],
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
		keys[emergencytypes.StoreKey],
		app.LegibleStakingKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.LegibleGovKeeper = lgmgovkeeper.NewKeeper(
		appCodec,
		keys[lgmgovtypes.StoreKey],
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		NewStakeQueryAdapter(app.LegibleStakingKeeper),
		app.BankKeeper,
	)

	app.ResearchKeeper = researchkeeper.NewKeeper(
		appCodec,
		keys[researchtypes.StoreKey],
		app.BankKeeper,
		NewStakeQueryAdapter(app.LegibleStakingKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.VestingRewardsKeeper = vestingrewardskeeper.NewKeeper(
		appCodec,
		keys[vestingrewardstypes.StoreKey],
		app.BankKeeper,
		NewVestingStakingAdapter(app.LegibleStakingKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.OntologyKeeper = ontologykeeper.NewKeeper(
		appCodec,
		keys[ontologytypes.StoreKey],
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.EvidenceMgmtKeeper = evidencemgmtkeeper.NewKeeper(
		appCodec,
		keys[evidencemgmttypes.StoreKey],
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.ClaimingPotKeeper = claimingpotkeeper.NewKeeper(
		appCodec,
		keys[claimingpottypes.StoreKey],
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.ChannelsKeeper = channelskeeper.NewKeeper(
		appCodec,
		keys[channelstypes.StoreKey],
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.TreeKeeper = treekeeper.NewKeeper(
		appCodec,
		keys[treetypes.StoreKey],
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.BVMKeeper = bvmkeeper.NewKeeper(
		appCodec,
		keys[bvmtypes.StoreKey],
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.QualificationKeeper = qualificationkeeper.NewKeeper(
		appCodec,
		keys[qualificationtypes.StoreKey],
		app.OntologyKeeper,
		app.LegibleStakingKeeper,
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.CaptureDefenseKeeper = capturedefensekeeper.NewKeeper(
		appCodec,
		keys[capturedefensetypes.StoreKey],
		NewStakingKeeperAdapter(app.LegibleStakingKeeper),
		NewCaptureDefenseOntologyAdapter(app.OntologyKeeper),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.CaptureChallengeKeeper = capturechallengekeeper.NewKeeper(
		appCodec,
		keys[capturechallengetypes.StoreKey],
		app.CaptureDefenseKeeper,
		NewStakingKeeperAdapter(app.LegibleStakingKeeper),
		app.BankKeeper,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.HomeKeeper = homekeeper.NewKeeper(
		appCodec,
		keys[hometypes.StoreKey],
		app.BankKeeper,
		nil, // tokensKeeper — nil-safe, wire when needed
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	app.KnowledgeKeeper = knowledgekeeper.NewKeeper(
		appCodec,
		keys[knowledgetypes.StoreKey],
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

	// Wire ontology keeper into knowledge for dynamic stratum ceilings
	app.KnowledgeKeeper.SetOntologyKeeper(app.OntologyKeeper)

	// Wire cross-module keepers into BVM for opcode host functions
	app.BVMKeeper.SetKnowledgeKeeper(NewBVMKnowledgeAdapter(app.KnowledgeKeeper))
	app.BVMKeeper.SetStakingKeeper(NewBVMStakingAdapter(app.LegibleStakingKeeper))
	app.BVMKeeper.SetAuthKeeper(NewBVMAuthAdapter(app.LegibleAuthKeeper))

	app.DisputesKeeper = disputeskeeper.NewKeeper(
		appCodec,
		keys[disputestypes.StoreKey],
		app.BankKeeper,
		NewDisputeKnowledgeAdapter(app.KnowledgeKeeper),
		NewDisputeStakingAdapter(app.LegibleStakingKeeper),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	app.DisputesKeeper.SetVestingKeeper(NewDisputeVestingAdapter(app.VestingRewardsKeeper))

	// ---- Gov Keeper (depends on other keepers for proposal handlers) ----

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
		AddRoute(govtypes.RouterKey, govv1beta1.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(app.ParamsKeeper))

	govKeeper.SetLegacyRouter(govRouter)

	// ---- IBC Router ----

	ibcRouter := ibcporttypes.NewRouter()
	ibcRouter.AddRoute(ibctransfertypes.ModuleName, ibctransfer.NewIBCModule(app.TransferKeeper))
	// Custom IBC modules (Phase 2: knowledge oracle IBC module)
	// ibcRouter.AddRoute("knowledge", knowledgeIBCModule)
	app.IBCKeeper.SetRouter(ibcRouter)

	// ---- Module Manager ----

	app.ModuleManager = module.NewManager(
		genutil.NewAppModule(app.AccountKeeper, app.StakingKeeper, app, app.txConfig),
		auth.NewAppModule(appCodec, app.AccountKeeper, nil, app.GetSubspace(authtypes.ModuleName)),
		vesting.NewAppModule(app.AccountKeeper, app.BankKeeper),
		bank.NewAppModule(appCodec, app.BankKeeper, app.AccountKeeper, app.GetSubspace(banktypes.ModuleName)),
		crisis.NewAppModule(app.CrisisKeeper, cast.ToBool(appOpts.Get("skip-crisis-invariants")), app.GetSubspace(crisistypes.ModuleName)),
		feegrantmodule.NewAppModule(appCodec, app.AccountKeeper, app.BankKeeper, app.FeeGrantKeeper, app.interfaceRegistry),
		gov.NewAppModule(appCodec, app.GovKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(govtypes.ModuleName)),
		mint.NewAppModule(appCodec, app.MintKeeper, app.AccountKeeper, nil, app.GetSubspace(minttypes.ModuleName)),
		slashing.NewAppModule(appCodec, app.SlashingKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(slashingtypes.ModuleName), app.interfaceRegistry),
		distr.NewAppModule(appCodec, app.DistrKeeper, app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.GetSubspace(distrtypes.ModuleName)),
		staking.NewAppModule(appCodec, app.StakingKeeper, app.AccountKeeper, app.BankKeeper, app.GetSubspace(stakingtypes.ModuleName)),
		upgrade.NewAppModule(app.UpgradeKeeper, app.AccountKeeper.AddressCodec()),
		evidence.NewAppModule(app.EvidenceKeeper),
		params.NewAppModule(app.ParamsKeeper),
		consensus.NewAppModule(appCodec, app.ConsensusKeeper),
		ibc.NewAppModule(app.IBCKeeper),
		ibctransfer.NewAppModule(app.TransferKeeper),
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
		treemodule.NewAppModule(appCodec, app.TreeKeeper),
		bvmmodule.NewAppModule(appCodec, app.BVMKeeper),
		homemodule.NewAppModule(appCodec, app.HomeKeeper),
		qualificationmodule.NewAppModule(appCodec, app.QualificationKeeper),
		capturedefensemodule.NewAppModule(appCodec, app.CaptureDefenseKeeper),
		capturechallengemodule.NewAppModule(appCodec, app.CaptureChallengeKeeper),
	)

	// Module init order — standard Cosmos SDK ordering.
	app.ModuleManager.SetOrderBeginBlockers(
		upgradetypes.ModuleName,
		minttypes.ModuleName,
		distrtypes.ModuleName,
		slashingtypes.ModuleName,
		evidencetypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		// Custom Legible Money modules
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
		treetypes.ModuleName,               // Expire seeds, check task deadlines
		bvmtypes.ModuleName,                // Execute pending contract schedules
		hometypes.ModuleName,               // Deadman switches, session cleanup
		qualificationtypes.ModuleName,      // Promote probationary, expire stale qualifications
		capturedefensetypes.ModuleName,     // Expire qualifications, decay reputations
		capturechallengetypes.ModuleName,   // Advance challenge phases
		knowledgetypes.ModuleName,
	)

	app.ModuleManager.SetOrderEndBlockers(
		crisistypes.ModuleName,
		govtypes.ModuleName,
		stakingtypes.ModuleName,
		ibcexported.ModuleName,
		ibctransfertypes.ModuleName,
		feegrant.ModuleName,
		// Custom Legible Money modules
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
		treetypes.ModuleName,
		bvmtypes.ModuleName,
		hometypes.ModuleName,               // Cleanup old alerts
		qualificationtypes.ModuleName,
		capturedefensetypes.ModuleName,     // Periodic capture detection
		capturechallengetypes.ModuleName,   // Auto-fund bounty pools
	)

	app.ModuleManager.SetOrderInitGenesis(
		authtypes.ModuleName,
		banktypes.ModuleName,
		distrtypes.ModuleName,
		stakingtypes.ModuleName,
		slashingtypes.ModuleName,
		govtypes.ModuleName,
		minttypes.ModuleName,
		crisistypes.ModuleName,
		ibcexported.ModuleName,
		genutiltypes.ModuleName,
		evidencetypes.ModuleName,
		ibctransfertypes.ModuleName,
		feegrant.ModuleName,
		paramstypes.ModuleName,
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
		treetypes.ModuleName,
		bvmtypes.ModuleName,
		hometypes.ModuleName,
		qualificationtypes.ModuleName,
		capturedefensetypes.ModuleName,
		capturechallengetypes.ModuleName,
		knowledgetypes.ModuleName,
	)

	app.ModuleManager.RegisterInvariants(app.CrisisKeeper)
	app.configurator = module.NewConfigurator(appCodec, bApp.MsgServiceRouter(), bApp.GRPCQueryRouter())
	if err = app.ModuleManager.RegisterServices(app.configurator); err != nil {
		panic(err)
	}

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

// GetSubspace returns a param subspace for a given module name.
func (app *LegibleApp) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := app.ParamsKeeper.GetSubspace(moduleName)
	return subspace
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
func (app *LegibleApp) prepForZeroHeightGenesis(ctx sdk.Context, jailAllowedAddrs []string) {
	// Standard Cosmos SDK zero-height genesis prep.
	// For Phase 1 MVC, this is a no-op stub.
	// Full implementation will handle:
	// - Withdraw all delegator rewards
	// - Reset slashing info
	// - Reset delegation heights
	// - Reset validator bond heights
	_ = ctx
	_ = jailAllowedAddrs
}

// ---------- Helpers ----------

func initParamsKeeper(appCodec codec.Codec, legacyAmino *codec.LegacyAmino, key *storetypes.KVStoreKey, tkey *storetypes.TransientStoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName)
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(ibcexported.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	// Custom Legible Money module param subspaces
	paramsKeeper.Subspace(knowledgetypes.ModuleName)
	paramsKeeper.Subspace(legiblestakingtypes.ModuleName)

	return paramsKeeper
}

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
