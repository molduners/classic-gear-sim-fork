package core

import (
	"fmt"
	"math"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	goproto "google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/core/simsignals"
)

const (
	defaultIterationsPerCombo = 1000
)

// raidSimRunner runs a standard raid simulation.
type raidSimRunner func(*proto.RaidSimRequest, chan *proto.ProgressMetrics, bool, simsignals.Signals) *proto.RaidSimResult

// bulkSimRunner runs a bulk simulation.
type bulkSimRunner struct {
	// SingleRaidSimRunner used to run one simulation of the bulk.
	SingleRaidSimRunner raidSimRunner
	// Request used for this bulk simulation.
	Request *proto.BulkSimRequest
}

func BulkSim(signals simsignals.Signals, request *proto.BulkSimRequest, progress chan *proto.ProgressMetrics) *proto.BulkSimResult {
	bulk := &bulkSimRunner{
		SingleRaidSimRunner: runSim,
		Request:             request,
	}

	result := bulk.Run(signals, progress)

	if progress != nil {
		progress <- &proto.ProgressMetrics{
			FinalBulkResult: result,
		}
		close(progress)
	}

	return result
}

type singleBulkSim struct {
	req *proto.RaidSimRequest
	cl  *raidSimRequestChangeLog
	eq  *equipmentSubstitution
}

func (b *bulkSimRunner) Run(signals simsignals.Signals, progress chan *proto.ProgressMetrics) (result *proto.BulkSimResult) {
	defer func() {
		if err := recover(); err != nil {
			result = &proto.BulkSimResult{
				Error: &proto.ErrorOutcome{Message: fmt.Sprintf("%v\nStack Trace:\n%s", err, string(debug.Stack()))},
			}
		}
		signals.Abort.Trigger()
	}()

	// Bulk simming is only supported for the single-player use (i.e. not whole raid-wide simming).
	// Verify that we have exactly 1 player.
	var playerCount int
	var player *proto.Player
	for _, p := range b.Request.GetBaseSettings().GetRaid().GetParties() {
		for _, pl := range p.GetPlayers() {
			// TODO(Riotdog-GehennasEU): Better way to check if a player is valid/set?
			if pl.Name != "" {
				player = pl
				playerCount++
			}
		}
	}
	if playerCount != 1 || player == nil {
		return &proto.BulkSimResult{
			Error: &proto.ErrorOutcome{
				Message: fmt.Sprintf("bulksim: expected exactly 1 player, found %d", playerCount),
			},
		}
	}
	if player.GetDatabase() != nil {
		addToDatabase(player.GetDatabase())
	}
	// reduce to just base party.
	b.Request.BaseSettings.Raid.Parties = []*proto.Party{b.Request.BaseSettings.Raid.Parties[0]}
	// clean to reduce memory
	player.Database = nil

	// Gemming for now can happen before slots are decided.
	// We might have to add logic after slot decisions if we want to enforce keeping meta gem active.

	iterations := b.Request.GetBulkSettings().GetIterationsPerCombo()
	if iterations <= 0 {
		iterations = defaultIterationsPerCombo
	}

	items := b.Request.GetBulkSettings().GetItems()
	// numItems := len(items)
	// if b.Request.BulkSettings.Combinations && numItems > maxItemCount {
	// 	return nil, fmt.Errorf("too many items specified (%d > %d), not computationally feasible", numItems, maxItemCount)
	// }

	// Create all distinct combinations of (item, slot). For example, let's say the only item we
	// want to bulk sim is a one-handed item that can be worn both as an off-hand or a main-hand weapon.
	// For each slot, we will create one itemWithSlot pair, so (item, off-hand) and (item, main-hand).
	// We verify later that we are not emitting any invalid equipment set.
	var distinctItemSlotCombos []*itemWithSlot
	for index, is := range items {
		item, ok := ItemsByID[is.Id]
		if !ok {
			return &proto.BulkSimResult{
				Error: &proto.ErrorOutcome{
					Message: fmt.Sprintf("unknown item with id %d in bulk settings", is.Id),
				},
			}
		}
		for _, slot := range eligibleSlotsForItem(item) {
			distinctItemSlotCombos = append(distinctItemSlotCombos, &itemWithSlot{
				Item:  is,
				Slot:  slot,
				Index: index,
			})
		}
	}
	baseItems := player.Equipment.Items

	allCombos := generateAllEquipmentSubstitutions(signals, baseItems, b.Request.BulkSettings.Combinations, distinctItemSlotCombos)

	var baseCombo *singleBulkSim
	var validCombos []singleBulkSim
	count := 0
	for sub := range allCombos {
		count++
		if count > 1000000 {
			panic("over 1 million combos, abandoning attempt")
		}
		substitutedRequest, changeLog := createNewRequestWithSubstitution(b.Request.BaseSettings, sub, b.Request.BulkSettings.AutoEnchant)
		combo := singleBulkSim{req: substitutedRequest, cl: changeLog, eq: sub}
		if !sub.HasItemReplacements() {
			// The equipped gear. Every other combo is measured against it.
			baseCombo = &combo
		} else if isValidEquipment(substitutedRequest.Raid.Parties[0].Players[0].Equipment) {
			validCombos = append(validCombos, combo)
		}
	}
	if baseCombo == nil {
		return &proto.BulkSimResult{
			Error: &proto.ErrorOutcome{
				Message: "no base result for equipped gear found in bulk sim",
			},
		}
	}

	// TODO(Riotdog-GehennasEU): Make this configurable?
	maxResults := 30

	newIters := int64(iterations)
	if b.Request.BulkSettings.FastMode {
		newIters /= 100

		// In fast mode try to keep starting iterations between 50 and 1000.
		newIters = min(max(newIters, 50), 1000, int64(iterations))
	}

	maxIterations := newIters * int64(len(validCombos)+1)
	if maxIterations > math.MaxInt32 {
		return &proto.BulkSimResult{
			Error: &proto.ErrorOutcome{Message: fmt.Sprintf("number of total iterations %d too large", maxIterations)},
		}
	}

	// Fast mode sims every combo with few iterations, drops the ones that are clearly worse than
	// the top results, and repeats with more iterations until the survivors get the full count.
	var rankedResults []*itemSubstitutionSimResult
	var baseResult *itemSubstitutionSimResult
	for {
		var errorOutcome *proto.ErrorOutcome
		rankedResults, baseResult, errorOutcome = b.getRankedResults(signals, *baseCombo, validCombos, newIters, progress)
		if errorOutcome != nil {
			return &proto.BulkSimResult{Error: errorOutcome}
		}

		if !b.Request.BulkSettings.FastMode || newIters >= int64(iterations) {
			break
		}

		validCombos = pruneBulkCombos(rankedResults, maxResults)
		if len(validCombos) <= maxResults {
			newIters = int64(iterations)
		} else {
			newIters = min(newIters*2, int64(iterations))
		}
	}

	// The equipped gear is listed among the results too, wherever it ranks.
	rankedResults = append(rankedResults, baseResult)
	sort.SliceStable(rankedResults, func(i, j int) bool {
		return rankedResults[i].Score() > rankedResults[j].Score()
	})
	if len(rankedResults) > maxResults {
		rankedResults = rankedResults[:maxResults]
	}

	result = &proto.BulkSimResult{
		EquippedGearResult: baseResult.toProto(int32(newIters)),
	}
	for _, r := range rankedResults {
		result.Results = append(result.Results, r.toProto(int32(newIters)))
	}

	if progress != nil {
		progress <- &proto.ProgressMetrics{
			FinalBulkResult: result,
		}
	}

	return result
}

// pruneBulkCombos keeps the combos that could still make the top maxResults: those whose gap to
// the maxResults-th best is within 3 standard errors. ranked must be sorted best first.
func pruneBulkCombos(ranked []*itemSubstitutionSimResult, maxResults int) []singleBulkSim {
	cutoff := ranked[min(maxResults, len(ranked))-1]
	var kept []singleBulkSim
	for _, r := range ranked {
		gap := cutoff.DpsDelta - r.DpsDelta
		noise := math.Sqrt(r.DpsDeltaStderr*r.DpsDeltaStderr + cutoff.DpsDeltaStderr*cutoff.DpsDeltaStderr)
		if gap <= 3*noise {
			kept = append(kept, singleBulkSim{req: r.Request, cl: r.ChangeLog, eq: r.Substitution})
		}
	}
	return kept
}

// getRankedResults sims the equipped gear, then every combo with the same iterations and random
// seed, and measures each combo's DPS gain over the equipped gear run by run. Returns the combos
// sorted best first, and the equipped gear's result.
func (b *bulkSimRunner) getRankedResults(signals simsignals.Signals, base singleBulkSim, validCombos []singleBulkSim, iterations int64, progress chan *proto.ProgressMetrics) ([]*itemSubstitutionSimResult, *itemSubstitutionSimResult, *proto.ErrorOutcome) {
	concurrency := runtime.NumCPU() + 1
	if concurrency <= 0 {
		concurrency = 2
	}

	tickets := make(chan struct{}, concurrency)
	for i := 0; i < concurrency; i++ {
		tickets <- struct{}{}
	}

	results := make(chan *itemSubstitutionSimResult, 10)

	numCombinations := int32(len(validCombos) + 1)
	totalIterationsUpperBound := int64(numCombinations) * iterations

	var totalCompletedIterations int32
	var totalCompletedSims int32

	reporterSignal := simsignals.CreateSignals()
	defer reporterSignal.Abort.Trigger() // cancel reporter

	// reporter for all sims combined.
	go func() {
		for !signals.Abort.IsTriggered() && !reporterSignal.Abort.IsTriggered() {
			complIters := atomic.LoadInt32(&totalCompletedIterations)
			complSims := atomic.LoadInt32(&totalCompletedSims)

			// stop reporting
			if complIters == int32(totalIterationsUpperBound) || numCombinations == complSims {
				return
			}

			if progress != nil {
				progress <- &proto.ProgressMetrics{
					TotalSims:           numCombinations,
					CompletedSims:       complSims,
					CompletedIterations: complIters,
					TotalIterations:     int32(totalIterationsUpperBound),
				}
			}
			time.Sleep(time.Second)
		}
	}()

	runCombo := func(sub singleBulkSim) *itemSubstitutionSimResult {
		singleSimProgress := make(chan *proto.ProgressMetrics)
		// watches this progress and pushes up to main reporter.
		go func() {
			var prevDone int32
			for p := range singleSimProgress {
				delta := p.CompletedIterations - prevDone
				atomic.AddInt32(&totalCompletedIterations, delta)
				prevDone = p.CompletedIterations
				if p.FinalRaidResult != nil {
					break
				}
			}
		}()
		// overwrite the requests iterations with the input for this function.
		sub.req.SimOptions.Iterations = int32(iterations)
		// Keep each iteration's DPS so combos can be compared with the equipped gear run by run.
		sub.req.SimOptions.SaveAllValues = true
		result := &itemSubstitutionSimResult{
			Request:      sub.req,
			Result:       b.SingleRaidSimRunner(sub.req, singleSimProgress, false, signals),
			Substitution: sub.eq,
			ChangeLog:    sub.cl,
		}
		atomic.AddInt32(&totalCompletedSims, 1)
		return result
	}

	baseResult := runCombo(base)
	if baseResult.Result == nil || baseResult.Result.Error != nil {
		return nil, nil, baseResult.Result.GetError()
	}
	baseValues := baseResult.playerDps().GetAllValues()

	// launcher for all combos (limited by concurrency max)
	go func() {
		for _, singleCombo := range validCombos {
			<-tickets
			go func(sub singleBulkSim) {
				results <- runCombo(sub)
				tickets <- struct{}{} // when done, allow for new sim to be launched.
			}(singleCombo)
		}
	}()

	rankedResults := make([]*itemSubstitutionSimResult, len(validCombos))
	for i := range rankedResults {
		result := <-results
		if result.Result == nil || result.Result.Error != nil {
			return nil, nil, result.Result.GetError()
		}
		result.DpsDelta, result.DpsDeltaStderr = pairedDifference(result.playerDps().GetAllValues(), baseValues)
		clearAllValues(result.Result.ProtoReflect())
		rankedResults[i] = result
	}
	clearAllValues(baseResult.Result.ProtoReflect())

	sort.Slice(rankedResults, func(i, j int) bool {
		return rankedResults[i].Score() > rankedResults[j].Score()
	})
	return rankedResults, baseResult, nil
}

// pairedDifference returns the mean of a[i]-b[i] and its standard error. Runs with the same index
// used the same random seed, so pairing them cancels much of the noise both share.
func pairedDifference(a, b []float64) (mean float64, stderr float64) {
	n := min(len(a), len(b))
	if n == 0 {
		return 0, 0
	}
	for i := 0; i < n; i++ {
		mean += a[i] - b[i]
	}
	mean /= float64(n)
	if n == 1 {
		return mean, 0
	}
	var sumSq float64
	for i := 0; i < n; i++ {
		d := a[i] - b[i] - mean
		sumSq += d * d
	}
	return mean, math.Sqrt(sumSq / float64(n-1) / float64(n))
}

// clearAllValues drops the per-iteration values from every distribution in a result, so a large
// bulk sim doesn't hold them all in memory.
func clearAllValues(m protoreflect.Message) {
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.Name() == "all_values":
			m.Clear(fd)
		case fd.Kind() != protoreflect.MessageKind:
		case fd.IsList():
			for i := 0; i < v.List().Len(); i++ {
				clearAllValues(v.List().Get(i).Message())
			}
		case fd.IsMap():
			v.Map().Range(func(_ protoreflect.MapKey, mv protoreflect.Value) bool {
				if fd.MapValue().Kind() == protoreflect.MessageKind {
					clearAllValues(mv.Message())
				}
				return true
			})
		default:
			clearAllValues(v.Message())
		}
		return true
	})
}

// itemSubstitutionSimResult stores the request and response of a simulation, along with the used
// equipment susbstitution and a changelog of which items were added and removed from the base
// equipment set.
type itemSubstitutionSimResult struct {
	Request      *proto.RaidSimRequest
	Result       *proto.RaidSimResult
	Substitution *equipmentSubstitution
	ChangeLog    *raidSimRequestChangeLog

	// DPS gained over the equipped gear, and its standard error.
	DpsDelta       float64
	DpsDeltaStderr float64
}

func (r *itemSubstitutionSimResult) playerDps() *proto.DistributionMetrics {
	return r.Result.GetRaidMetrics().GetParties()[0].GetPlayers()[0].GetDps()
}

func (r *itemSubstitutionSimResult) toProto(iterations int32) *proto.BulkComboResult {
	um := r.Result.GetRaidMetrics().GetParties()[0].GetPlayers()[0]
	um.Actions = nil
	um.Auras = nil
	um.Resources = nil
	um.Pets = nil
	return &proto.BulkComboResult{
		ItemsAdded:     r.ChangeLog.AddedItems,
		UnitMetrics:    um,
		DpsDelta:       r.DpsDelta,
		DpsDeltaStderr: r.DpsDeltaStderr,
		Iterations:     iterations,
	}
}

// Score used to rank results.
func (r *itemSubstitutionSimResult) Score() float64 {
	if r.Result == nil || r.Result.Error != nil {
		return 0
	}
	return r.Result.RaidMetrics.Dps.Avg
}

// equipmentSubstitution specifies all items to be used as replacements for the equipped gear.
type equipmentSubstitution struct {
	Items []*itemWithSlot
}

// usesItem returns true if this exact bag item is already used in another slot.
func (es *equipmentSubstitution) usesItem(item *proto.ItemSpec) bool {
	for _, repl := range es.Items {
		if repl.Item == item {
			return true
		}
	}
	return false
}

// HasChanges returns true if the equipment substitution has any item replacmenets.
func (es *equipmentSubstitution) HasItemReplacements() bool {
	return len(es.Items) > 0
}

func (es *equipmentSubstitution) CanonicalHash() string {
	slotToID := map[proto.ItemSlot]int32{}
	for _, repl := range es.Items {
		slotToID[repl.Slot] = repl.Item.Id
	}

	// Canonical representation always has the ring or trinket with smaller item ID in slot1
	// if the equipment substitution mentions two rings or trinkets.
	if ring1, ok := slotToID[proto.ItemSlot_ItemSlotFinger1]; ok {
		if ring2, ok := slotToID[proto.ItemSlot_ItemSlotFinger2]; ok {
			if ring1 == ring2 {
				return ""
			}
			if ring1 > ring2 {
				slotToID[proto.ItemSlot_ItemSlotFinger1], slotToID[proto.ItemSlot_ItemSlotFinger2] = ring2, ring1
			}
		}
	}
	if trink1, ok := slotToID[proto.ItemSlot_ItemSlotTrinket1]; ok {
		if trink2, ok := slotToID[proto.ItemSlot_ItemSlotTrinket2]; ok {
			if trink1 == trink2 {
				return ""
			}
			if trink1 > trink2 {
				slotToID[proto.ItemSlot_ItemSlotTrinket1], slotToID[proto.ItemSlot_ItemSlotTrinket2] = trink2, trink1
			}
		}
	}

	parts := make([]string, 0, len(proto.ItemSlot_name))
	for i := 0; i < len(proto.ItemSlot_name); i++ {
		if id, ok := slotToID[proto.ItemSlot(i)]; ok {
			parts = append(parts, fmt.Sprintf("%d=%d", i, id))
		}
	}

	return strings.Join(parts, ":")
}

// isValidEquipment returns true if the specified equipment spec is valid. An equipment spec
// is valid if it does not reference a two-hander and off-hand weapon combo.
func isValidEquipment(equipment *proto.EquipmentSpec) bool {
	var usesTwoHander, usesOffhand bool

	// Validate weapons
	if knownItem, ok := ItemsByID[equipment.Items[proto.ItemSlot_ItemSlotMainHand].Id]; ok {
		usesTwoHander = knownItem.HandType == proto.HandType_HandTypeTwoHand
	}
	if knownItem, ok := ItemsByID[equipment.Items[proto.ItemSlot_ItemSlotOffHand].Id]; ok {
		usesOffhand = knownItem.HandType == proto.HandType_HandTypeOffHand
	}
	if usesTwoHander && usesOffhand {
		return false
	}

	// Validate trinkets/rings for duplicate IDs
	if equipment.Items[proto.ItemSlot_ItemSlotFinger1].Id == equipment.Items[proto.ItemSlot_ItemSlotFinger2].Id && equipment.Items[proto.ItemSlot_ItemSlotFinger1].Id != 0 {
		return false
	} else if equipment.Items[proto.ItemSlot_ItemSlotTrinket1].Id == equipment.Items[proto.ItemSlot_ItemSlotTrinket2].Id && equipment.Items[proto.ItemSlot_ItemSlotTrinket1].Id != 0 {
		return false
	}

	// Validate rings/trinkets for heroic/non-heroic (matching name)
	f1, ok1 := ItemsByID[equipment.Items[proto.ItemSlot_ItemSlotFinger1].Id]
	f2, ok2 := ItemsByID[equipment.Items[proto.ItemSlot_ItemSlotFinger2].Id]
	if ok1 && ok2 && f1.Name == f2.Name {
		return false
	}

	t1, ok1 := ItemsByID[equipment.Items[proto.ItemSlot_ItemSlotTrinket1].Id]
	t2, ok2 := ItemsByID[equipment.Items[proto.ItemSlot_ItemSlotTrinket2].Id]
	if ok1 && ok2 && t1.Name == t2.Name {
		return false
	}

	return true
}

// generateAllEquipmentSubstitutions generates all possible valid equipment substitutions for the
// given bulk sim request. Also returns the unchanged equipment ("base equipment set") set as the
// first result. This ensures that simming over all possible equipment substitutions includes the
// base case as well.
func generateAllEquipmentSubstitutions(signals simsignals.Signals, baseItems []*proto.ItemSpec, combinations bool, distinctItemSlotCombos []*itemWithSlot) chan *equipmentSubstitution {
	results := make(chan *equipmentSubstitution)
	go func() {
		defer close(results)

		// No substitutions (base case).
		results <- &equipmentSubstitution{}

		// Organize everything by slot.
		itemsBySlot := make([][]*proto.ItemSpec, 17)
		for _, is := range distinctItemSlotCombos {
			itemsBySlot[is.Slot] = append(itemsBySlot[is.Slot], is.Item)
		}

		if !combinations {
			// seenCombos lets us deduplicate trinket/ring combos.
			comboChecker := ItemComboChecker{}

			// Pre-seed the existing item combos
			comboChecker.HasCombo(baseItems[proto.ItemSlot_ItemSlotFinger1].Id, baseItems[proto.ItemSlot_ItemSlotFinger2].Id)
			comboChecker.HasCombo(baseItems[proto.ItemSlot_ItemSlotTrinket1].Id, baseItems[proto.ItemSlot_ItemSlotTrinket2].Id)

			for slotid, slot := range itemsBySlot {
				for _, item := range slot {
					sub := equipmentSubstitution{
						Items: []*itemWithSlot{{Item: item, Slot: proto.ItemSlot(slotid)}},
					}
					// Handle finger/trinket specially to generate combos
					switch slotid {
					case int(proto.ItemSlot_ItemSlotFinger1), int(proto.ItemSlot_ItemSlotTrinket1):
						if !comboChecker.HasCombo(item.Id, baseItems[slotid+1].Id) {
							results <- &sub
						}
						// Generate extra combos
						subslot := slotid + 1
						for _, subitem := range itemsBySlot[subslot] {
							if shouldSkipCombo(baseItems, subitem, proto.ItemSlot(subslot), comboChecker, sub) {
								continue
							}
							miniCombo := createReplacement(sub, &itemWithSlot{Item: subitem, Slot: proto.ItemSlot(subslot)})
							results <- &miniCombo
						}
					case int(proto.ItemSlot_ItemSlotFinger2), int(proto.ItemSlot_ItemSlotTrinket2):
						// Ensure we don't have this combo with the base equipment.
						if !comboChecker.HasCombo(item.Id, baseItems[slotid-1].Id) {
							results <- &sub
						}
					default:
						results <- &sub
					}
				}
			}
			return
		}

		// Simming all combinations of items. This is useful to find the e.g.
		// the best set of items in your bags.
		subComboChecker := SubstitutionComboChecker{}
		for i := 0; i < len(itemsBySlot); i++ {
			genSlotCombos(proto.ItemSlot(i), baseItems, equipmentSubstitution{}, itemsBySlot, subComboChecker, results)
		}
	}()

	return results
}

func createReplacement(repl equipmentSubstitution, item *itemWithSlot) equipmentSubstitution {
	newItems := make([]*itemWithSlot, len(repl.Items))
	copy(newItems, repl.Items)
	newItems = append(newItems, item)
	repl.Items = newItems
	return repl
}

func shouldSkipCombo(baseItems []*proto.ItemSpec, item *proto.ItemSpec, slot proto.ItemSlot, comboChecker ItemComboChecker, replacements equipmentSubstitution) bool {
	switch slot {
	case proto.ItemSlot_ItemSlotFinger1, proto.ItemSlot_ItemSlotTrinket1:
		return comboChecker.HasCombo(item.Id, baseItems[slot+1].Id)
	case proto.ItemSlot_ItemSlotFinger2, proto.ItemSlot_ItemSlotTrinket2:

		for _, repl := range replacements.Items {
			if slot == proto.ItemSlot_ItemSlotFinger2 && repl.Slot == proto.ItemSlot_ItemSlotFinger1 ||
				slot == proto.ItemSlot_ItemSlotTrinket2 && repl.Slot == proto.ItemSlot_ItemSlotTrinket1 {
				return comboChecker.HasCombo(repl.Item.Id, item.Id)
			}
		}
		// Since we didn't find an item in the opposite slot, check against base items.
		return comboChecker.HasCombo(item.Id, baseItems[slot-1].Id)
	}
	return false
}

func genSlotCombos(slot proto.ItemSlot, baseItems []*proto.ItemSpec, baseRepl equipmentSubstitution, replaceBySlot [][]*proto.ItemSpec, comboChecker SubstitutionComboChecker, results chan *equipmentSubstitution) {
	// Iterate all items in this slot, add to the baseRepl, then descend to add all other item combos.
	for _, item := range replaceBySlot[slot] {
		// A bag item can fit several slots (e.g. a one-hander in main or off hand), but there is only
		// one of it. Each bag item is a single *ItemSpec shared by all its slots, so compare pointers.
		if baseRepl.usesItem(item) {
			continue
		}
		// Create a new equipment substitution from the current replacements plus the new item.
		combo := createReplacement(baseRepl, &itemWithSlot{Slot: slot, Item: item})
		if comboChecker.HasCombo(combo) {
			continue
		}
		results <- &combo

		// Now descend to each other slot to pair with this combo.
		for j := slot + 1; int(j) < len(replaceBySlot); j++ {
			genSlotCombos(j, baseItems, combo, replaceBySlot, comboChecker, results)
		}
	}
}

// itemWithSlot pairs an item with its fixed item slot.
type itemWithSlot struct {
	Item *proto.ItemSpec
	Slot proto.ItemSlot

	// This index refers to the item's position in the BulkEquipmentSpec of the player and serves as
	// a unique item ID. It is used to verify that a valid equipmentSubstitution only references an
	// item once.
	Index int
}

// raidSimRequestChangeLog stores a change log of which items were added and removed from the base
// equipment set.
type raidSimRequestChangeLog struct {
	AddedItems []*proto.ItemSpecWithSlot
}

// createNewRequestWithSubstitution creates a copy of the input RaidSimRequest and applis the given
// equipment susbstitution to the player's equipment. Copies enchant if specified and possible.
func createNewRequestWithSubstitution(readonlyInputRequest *proto.RaidSimRequest, substitution *equipmentSubstitution, autoEnchant bool) (*proto.RaidSimRequest, *raidSimRequestChangeLog) {
	request := goproto.Clone(readonlyInputRequest).(*proto.RaidSimRequest)
	changeLog := &raidSimRequestChangeLog{}
	player := request.Raid.Parties[0].Players[0]
	equipment := player.Equipment
	for _, is := range substitution.Items {
		oldItem := equipment.Items[is.Slot]
		if autoEnchant && oldItem.Enchant > 0 && is.Item.Enchant == 0 {
			equipment.Items[is.Slot] = goproto.Clone(is.Item).(*proto.ItemSpec)
			equipment.Items[is.Slot].Enchant = oldItem.Enchant
			// TODO: logic to decide if the enchant can be applied to the new item...
			// Specifically, offhand shouldn't get shield enchant
			// Main/One hand shouldn't get staff enchant
			// Later: replace normal enchant if replacement is staff.

			changeLog.AddedItems = append(changeLog.AddedItems, &proto.ItemSpecWithSlot{
				Item: equipment.Items[is.Slot],
				Slot: is.Slot,
			})
		} else {
			equipment.Items[is.Slot] = is.Item
			changeLog.AddedItems = append(changeLog.AddedItems, &proto.ItemSpecWithSlot{
				Item: is.Item,
				Slot: is.Slot,
			})
		}
	}
	return request, changeLog
}

type ItemComboChecker map[int64]struct{}

func (ic *ItemComboChecker) HasCombo(itema int32, itemb int32) bool {
	if itema == itemb {
		return true
	}
	key := ic.generateComboKey(itema, itemb)
	if _, ok := (*ic)[key]; ok {
		return true
	} else {
		(*ic)[key] = struct{}{}
	}
	return false
}

// put this function on ic just so it isn't in global namespace
func (ic *ItemComboChecker) generateComboKey(itemA int32, itemB int32) int64 {
	// Item IDs are 32 bits, so shift by 32 to keep every pair distinct.
	if itemA > itemB {
		return int64(itemA) + int64(itemB)<<32
	}
	return int64(itemB) + int64(itemA)<<32
}

type SubstitutionComboChecker map[string]struct{}

func (ic *SubstitutionComboChecker) HasCombo(replacements equipmentSubstitution) bool {
	key := replacements.CanonicalHash()
	if key == "" {
		// Invalid combo.
		return true
	}
	if _, ok := (*ic)[key]; ok {
		return true
	}
	(*ic)[key] = struct{}{}
	return false
}
