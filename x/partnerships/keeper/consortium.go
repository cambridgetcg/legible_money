package keeper

import (
	"encoding/json"
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/partnerships/types"
)

// ---------- Key Construction Helpers ----------

func consortiumKey(id string) []byte {
	return append(types.ConsortiumKeyPrefix, []byte(id)...)
}

func consortiumFormationKey(id string) []byte {
	return append(types.ConsortiumFormationPrefix, []byte(id)...)
}

func byMemberPrefix(addr string) []byte {
	return append(types.ByMemberIndexPrefix, []byte(addr+"/")...)
}

func byMemberKey(addr, consortiumId string) []byte {
	return append(types.ByMemberIndexPrefix, []byte(addr+"/"+consortiumId)...)
}

// ---------- CRUD ----------

// SetConsortium stores a consortium and maintains member indexes.
func (k Keeper) SetConsortium(ctx sdk.Context, c *types.Consortium) {
	kvStore := k.storeService.OpenKVStore(ctx)

	bz, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal consortium: %v", err))
	}

	_ = kvStore.Set(consortiumKey(c.Id), bz)

	// Update member indexes
	for _, m := range c.Members {
		_ = kvStore.Set(byMemberKey(m.Address, c.Id), []byte(c.Id))
	}
}

// GetConsortium retrieves a consortium by ID.
func (k Keeper) GetConsortium(ctx sdk.Context, id string) (*types.Consortium, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(consortiumKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var c types.Consortium
	if err := json.Unmarshal(bz, &c); err != nil {
		return nil, false
	}
	return &c, true
}

// DeleteConsortium removes a consortium and its member indexes.
func (k Keeper) DeleteConsortium(ctx sdk.Context, c *types.Consortium) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(consortiumKey(c.Id))

	for _, m := range c.Members {
		_ = kvStore.Delete(byMemberKey(m.Address, c.Id))
	}
}

// GetAllConsortia returns all consortia.
func (k Keeper) GetAllConsortia(ctx sdk.Context) []*types.Consortium {
	var consortia []*types.Consortium
	k.IterateConsortia(ctx, func(c *types.Consortium) bool {
		consortia = append(consortia, c)
		return false
	})
	return consortia
}

// IterateConsortia iterates over all consortia. Return true to stop.
func (k Keeper) IterateConsortia(ctx sdk.Context, cb func(*types.Consortium) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.ConsortiumKeyPrefix, prefixEndBytes(types.ConsortiumKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var c types.Consortium
		if err := json.Unmarshal(iter.Value(), &c); err != nil {
			continue
		}
		if cb(&c) {
			break
		}
	}
}

// ---------- Index Queries ----------

// GetConsortiumsByMember returns all consortium IDs for a member address.
func (k Keeper) GetConsortiumsByMember(ctx sdk.Context, addr string) []string {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := byMemberPrefix(addr)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var ids []string
	for ; iter.Valid(); iter.Next() {
		ids = append(ids, string(iter.Value()))
	}
	return ids
}

// GetActiveConsortiumForAddress finds the first active consortium for an address.
func (k Keeper) GetActiveConsortiumForAddress(ctx sdk.Context, addr string) (*types.Consortium, bool) {
	for _, id := range k.GetConsortiumsByMember(ctx, addr) {
		if c, found := k.GetConsortium(ctx, id); found && c.Status == types.StatusActive {
			return c, true
		}
	}
	return nil, false
}

// ---------- Formation ----------

// CreateConsortium validates and stores a new consortium in "forming" status.
func (k Keeper) CreateConsortium(
	ctx sdk.Context,
	founder string,
	name string,
	memberAddrs []string,
	accountTypes []string,
	shareBps []uint64,
	lockTier uint32,
) (string, error) {
	if len(memberAddrs) < types.ConsortiumMinMembers {
		return "", types.ErrConsortiumTooSmall
	}
	if len(memberAddrs) > types.ConsortiumMaxMembers {
		return "", types.ErrConsortiumFull
	}
	if len(memberAddrs) != len(accountTypes) || len(memberAddrs) != len(shareBps) {
		return "", fmt.Errorf("memberAddrs, accountTypes, and shareBps must have same length")
	}
	if lockTier > 5 {
		return "", types.ErrInvalidLockTier
	}

	// Verify founder is in member list
	founderFound := false
	for _, addr := range memberAddrs {
		if addr == founder {
			founderFound = true
			break
		}
	}
	if !founderFound {
		return "", fmt.Errorf("founder must be in member list")
	}

	// Validate shares sum to 1M
	var totalShares uint64
	for _, s := range shareBps {
		totalShares += s
	}
	if totalShares != 1000000 {
		return "", fmt.Errorf("%w: got %d", types.ErrShareSumInvalid, totalShares)
	}

	currentBlock := uint64(ctx.BlockHeight())

	// Check no member is in active partnership or consortium
	for _, addr := range memberAddrs {
		if _, found := k.GetActivePartnershipForAddress(ctx, addr); found {
			return "", fmt.Errorf("%w: %s has active partnership", types.ErrMemberAlreadyInGroup, addr)
		}
		if _, found := k.GetActiveConsortiumForAddress(ctx, addr); found {
			return "", fmt.Errorf("%w: %s has active consortium", types.ErrMemberAlreadyInGroup, addr)
		}
	}

	// Build members
	members := make([]*types.ConsortiumMember, len(memberAddrs))
	for i, addr := range memberAddrs {
		role := "member"
		if addr == founder {
			role = "founder"
		}
		members[i] = &types.ConsortiumMember{
			Address:     addr,
			AccountType: accountTypes[i],
			ShareBps:    shareBps[i],
			JoinedAt:    currentBlock,
			Role:        role,
			Accepted:    addr == founder, // founder auto-accepts
		}
	}

	seq := k.NextSequence(ctx)
	consortiumId := fmt.Sprintf("consortium-%d", seq)

	lockDuration := types.LockTiers[lockTier].MinBlocks

	consortium := &types.Consortium{
		Id:                consortiumId,
		Name:              name,
		Status:            types.StatusForming,
		Members:           members,
		CommonPotBalance:  "0",
		TotalEarned:       "0",
		CommonPotShareBps: 220000, // default 22%, same as partnerships
		LockTier:          lockTier,
		LockExpiresAt:     currentBlock + lockDuration,
		FormedAtBlock:     0, // set when activated
		RequiredApprovals: uint32(len(members)),
		CooperationScore:  5000, // start at 50%
	}

	// Validate before storing
	if err := types.ValidateConsortium(consortium); err != nil {
		return "", err
	}

	k.SetConsortium(ctx, consortium)

	// Store formation expiry
	kvStore := k.storeService.OpenKVStore(ctx)
	formationExpiry := currentBlock + types.ConsortiumFormationWindow
	_ = kvStore.Set(
		consortiumFormationKey(consortiumId),
		[]byte(fmt.Sprintf("%d", formationExpiry)),
	)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"consortium_created",
			sdk.NewAttribute("consortium_id", consortiumId),
			sdk.NewAttribute("founder", founder),
			sdk.NewAttribute("member_count", fmt.Sprintf("%d", len(members))),
			sdk.NewAttribute("lock_tier", fmt.Sprintf("%d", lockTier)),
		),
	)

	k.Logger(ctx).Info("consortium formation initiated",
		"consortium_id", consortiumId,
		"founder", founder,
		"members", len(members),
	)

	return consortiumId, nil
}

// AcceptConsortium marks a member as accepted. If all accepted, activate.
func (k Keeper) AcceptConsortium(ctx sdk.Context, consortiumId string, acceptor string) error {
	consortium, found := k.GetConsortium(ctx, consortiumId)
	if !found {
		return types.ErrConsortiumNotFound
	}
	if consortium.Status != types.StatusForming {
		return fmt.Errorf("%w: consortium is %s", types.ErrInvalidStatus, consortium.Status)
	}

	// Check formation window
	currentBlock := uint64(ctx.BlockHeight())
	kvStore := k.storeService.OpenKVStore(ctx)
	expiryBz, err := kvStore.Get(consortiumFormationKey(consortiumId))
	if err != nil || expiryBz == nil {
		return types.ErrConsortiumFormationExp
	}
	var expiry uint64
	if _, err := fmt.Sscanf(string(expiryBz), "%d", &expiry); err != nil {
		return types.ErrConsortiumFormationExp
	}
	if currentBlock > expiry {
		return types.ErrConsortiumFormationExp
	}

	// Find member and mark accepted
	memberFound := false
	allAccepted := true
	for _, m := range consortium.Members {
		if m.Address == acceptor {
			if m.Accepted {
				return types.ErrAlreadyAccepted
			}
			m.Accepted = true
			memberFound = true
		}
		if !m.Accepted {
			allAccepted = false
		}
	}
	if !memberFound {
		return types.ErrNotConsortiumMember
	}

	// If all accepted, activate
	if allAccepted {
		consortium.Status = types.StatusActive
		consortium.FormedAtBlock = currentBlock

		// Clean up formation record
		_ = kvStore.Delete(consortiumFormationKey(consortiumId))
	}

	k.SetConsortium(ctx, consortium)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"consortium_accepted",
			sdk.NewAttribute("consortium_id", consortiumId),
			sdk.NewAttribute("acceptor", acceptor),
			sdk.NewAttribute("activated", fmt.Sprintf("%t", allAccepted)),
		),
	)

	return nil
}

// ---------- Reward Distribution ----------

// DistributeConsortiumReward splits a reward N-ways through consortium economics.
// Returns per-member shares and the amount added to pot.
func (k Keeper) DistributeConsortiumReward(
	ctx sdk.Context,
	consortium *types.Consortium,
	grossAmount string,
	sourceType string,
) (memberShares map[string]string, potAdd string, err error) {
	gross := new(big.Int)
	if _, ok := gross.SetString(grossAmount, 10); !ok || gross.Sign() <= 0 {
		return nil, "", fmt.Errorf("invalid gross amount: %s", grossAmount)
	}

	// Step 1: Apply lock multiplier by lock tier
	multiplier := GetLockMultiplier(consortium.LockTier)
	adjusted := new(big.Int).Mul(gross, multiplier)
	adjusted.Div(adjusted, bpsDenom)

	// Step 2: Calculate common pot share
	commonPotShareBps := new(big.Int).SetUint64(consortium.CommonPotShareBps)
	potShare := new(big.Int).Mul(adjusted, commonPotShareBps)
	potShare.Div(potShare, bpsDenom)

	// Step 3: Remaining split per member ShareBps
	remaining := new(big.Int).Sub(adjusted, potShare)

	memberShares = make(map[string]string)
	distributed := new(big.Int)

	for i, m := range consortium.Members {
		var share *big.Int
		if i == len(consortium.Members)-1 {
			// Last member gets remainder to avoid rounding loss
			share = new(big.Int).Sub(remaining, distributed)
		} else {
			memberBps := new(big.Int).SetUint64(m.ShareBps)
			share = new(big.Int).Mul(remaining, memberBps)
			share.Div(share, bpsDenom)
			distributed.Add(distributed, share)
		}
		memberShares[m.Address] = share.String()
	}

	// Step 4: Update consortium metrics
	currentPot := new(big.Int)
	if consortium.CommonPotBalance != "" {
		currentPot.SetString(consortium.CommonPotBalance, 10)
	}
	currentPot.Add(currentPot, potShare)
	consortium.CommonPotBalance = currentPot.String()

	currentEarned := new(big.Int)
	if consortium.TotalEarned != "" {
		currentEarned.SetString(consortium.TotalEarned, 10)
	}
	currentEarned.Add(currentEarned, adjusted)
	consortium.TotalEarned = currentEarned.String()

	k.SetConsortium(ctx, consortium)

	k.Logger(ctx).Info("consortium reward distributed",
		"consortium_id", consortium.Id,
		"gross", grossAmount,
		"adjusted", adjusted.String(),
		"common_pot", potShare.String(),
		"source", sourceType,
	)

	return memberShares, potShare.String(), nil
}

// ---------- Member Exit ----------

// HandleConsortiumMemberExit removes a member and redistributes shares.
func (k Keeper) HandleConsortiumMemberExit(ctx sdk.Context, consortiumId string, exitingAddr string) error {
	params := k.GetParams(ctx)
	currentBlock := uint64(ctx.BlockHeight())

	consortium, found := k.GetConsortium(ctx, consortiumId)
	if !found {
		return types.ErrConsortiumNotFound
	}

	if consortium.Status == types.StatusDissolved {
		return types.ErrAlreadyDissolved
	}
	if consortium.Status == types.StatusCooling {
		return types.ErrExitInProgress
	}

	// Check lock expiry (same rule as partnership: must be expired or suspended)
	if consortium.Status == types.StatusActive && consortium.LockExpiresAt > currentBlock {
		return types.ErrLockNotExpired
	}

	// Find exiting member
	exitIdx := -1
	for i, m := range consortium.Members {
		if m.Address == exitingAddr {
			exitIdx = i
			break
		}
	}
	if exitIdx == -1 {
		return types.ErrNotConsortiumMember
	}

	// Remove member index
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(byMemberKey(exitingAddr, consortiumId))

	// Remove member from list
	consortium.Members = append(consortium.Members[:exitIdx], consortium.Members[exitIdx+1:]...)

	// If below minimum, dissolve
	if len(consortium.Members) < types.ConsortiumMinMembers {
		cooldownEnd := currentBlock + params.CoolingPeriodBlocks
		consortium.Status = types.StatusCooling
		consortium.ExitState = &types.ConsortiumExitState{
			InitiatedBy: exitingAddr,
			InitiatedAt: currentBlock,
			CooldownEnd: cooldownEnd,
		}
		k.SetConsortium(ctx, consortium)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"consortium_dissolving",
				sdk.NewAttribute("consortium_id", consortiumId),
				sdk.NewAttribute("exiting_member", exitingAddr),
				sdk.NewAttribute("reason", "below_minimum_members"),
			),
		)
		return nil
	}

	// Redistribute shares proportionally among remaining members
	redistributeShares(consortium.Members)
	k.SetConsortium(ctx, consortium)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"consortium_member_exited",
			sdk.NewAttribute("consortium_id", consortiumId),
			sdk.NewAttribute("exiting_member", exitingAddr),
			sdk.NewAttribute("remaining_members", fmt.Sprintf("%d", len(consortium.Members))),
		),
	)

	return nil
}

// redistributeShares redistributes member shares proportionally so they sum to 1M.
// The last member gets the remainder to avoid rounding loss.
func redistributeShares(members []*types.ConsortiumMember) {
	if len(members) == 0 {
		return
	}

	// Sum existing shares
	var totalExisting uint64
	for _, m := range members {
		totalExisting += m.ShareBps
	}

	if totalExisting == 0 {
		// Equal split
		each := uint64(1000000) / uint64(len(members))
		for i, m := range members {
			if i == len(members)-1 {
				m.ShareBps = 1000000 - each*uint64(len(members)-1)
			} else {
				m.ShareBps = each
			}
		}
		return
	}

	// Proportional redistribution
	var distributed uint64
	for i, m := range members {
		if i == len(members)-1 {
			m.ShareBps = 1000000 - distributed
		} else {
			// share = oldShare * 1000000 / totalExisting
			m.ShareBps = m.ShareBps * 1000000 / totalExisting
			distributed += m.ShareBps
		}
	}
}

// ---------- Formation Expiry ----------

// ExpireConsortiumFormations dissolves consortia past their formation window.
func (k Keeper) ExpireConsortiumFormations(ctx sdk.Context) {
	currentBlock := uint64(ctx.BlockHeight())
	kvStore := k.storeService.OpenKVStore(ctx)

	iter, err := kvStore.Iterator(types.ConsortiumFormationPrefix, prefixEndBytes(types.ConsortiumFormationPrefix))
	if err != nil {
		return
	}

	var toDelete [][]byte
	for ; iter.Valid(); iter.Next() {
		var expiry uint64
		if _, err := fmt.Sscanf(string(iter.Value()), "%d", &expiry); err != nil {
			continue
		}
		if currentBlock > expiry {
			key := iter.Key()
			consortiumId := string(key[len(types.ConsortiumFormationPrefix):])
			if c, found := k.GetConsortium(ctx, consortiumId); found {
				if c.Status == types.StatusForming {
					c.Status = types.StatusDissolved
					k.SetConsortium(ctx, c)
				}
			}
			toDelete = append(toDelete, append([]byte{}, key...))
		}
	}
	iter.Close()
	for _, key := range toDelete {
		_ = kvStore.Delete(key)
	}
}

// ---------- Cooling Settlement ----------

// SettleCoolingConsortia dissolves consortia that have completed their cooling period.
func (k Keeper) SettleCoolingConsortia(ctx sdk.Context) {
	currentBlock := uint64(ctx.BlockHeight())

	k.IterateConsortia(ctx, func(c *types.Consortium) bool {
		if c.Status == types.StatusCooling && c.ExitState != nil && c.ExitState.CooldownEnd <= currentBlock {
			c.Status = types.StatusDissolved
			k.SetConsortium(ctx, c)

			k.Logger(ctx).Info("consortium dissolved after cooling",
				"consortium_id", c.Id,
			)
		}
		return false
	})
}

// ---------- Consortium Genesis ----------

// InitConsortiumGenesis initializes consortium state from a list of consortia.
func (k Keeper) InitConsortiumGenesis(ctx sdk.Context, consortia []*types.Consortium) {
	for _, c := range consortia {
		if c != nil {
			k.SetConsortium(ctx, c)
		}
	}
}

// ExportConsortiumGenesis exports all consortia for genesis export.
func (k Keeper) ExportConsortiumGenesis(ctx sdk.Context) []*types.Consortium {
	return k.GetAllConsortia(ctx)
}
