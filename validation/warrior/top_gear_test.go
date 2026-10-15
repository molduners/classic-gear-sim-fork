package warriorvalidation

import (
	"math"
	"testing"

	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"github.com/wowsims/classic/sim/ruleset"
	goproto "google.golang.org/protobuf/proto"
)

// Top Gear (the bulk sim) on the fury-dw-p1 warrior with a few phase 1 items in the bags.
// Bag: Mark of Fordring (neck), Devilsaur Leggings, Painweaver Band (ring), Spineshatter
// (one-hand mace, fits either hand) and Diamond Flask (trinket).
var topGearBag = []int32{15411, 15062, 13098, 19335, 20130}

func topGearRequest(t *testing.T, iterations int32, fastMode bool) *proto.BulkSimRequest {
	ruleset.Activate("vanilla")
	var file struct {
		Scenarios []scenario `json:"scenarios"`
	}
	loadJSON(t, "scenarios.json", &file)
	var base *proto.RaidSimRequest
	for _, sc := range file.Scenarios {
		if sc.Name == "fury-dw-p1" {
			base = buildRequest(t, sc, iterations)
		}
	}
	var items []*proto.ItemSpec
	for _, id := range topGearBag {
		items = append(items, &proto.ItemSpec{Id: id})
	}
	return &proto.BulkSimRequest{
		BaseSettings: base,
		BulkSettings: &proto.BulkSettings{Items: items, Combinations: true, FastMode: fastMode, IterationsPerCombo: iterations},
	}
}

func TestTopGear(t *testing.T) {
	const iterations = 500
	req := topGearRequest(t, iterations, false)
	baseSettings := goproto.Clone(req.BaseSettings).(*proto.RaidSimRequest)
	res := core.RunBulkSim(req)
	if res.Error != nil {
		t.Fatal(res.Error.Message)
	}
	equipped := res.EquippedGearResult
	if equipped.DpsDelta != 0 {
		t.Errorf("equipped gear gains %.2f DPS over itself, want 0", equipped.DpsDelta)
	}

	for rank, r := range res.Results {
		seen := map[int32]bool{}
		for _, added := range r.ItemsAdded {
			if seen[added.Item.Id] {
				t.Errorf("result %d uses bag item %d twice", rank, added.Item.Id)
			}
			seen[added.Item.Id] = true
		}
		if r.Iterations != iterations {
			t.Errorf("result %d ran %d iterations, want %d", rank, r.Iterations, iterations)
		}
	}

	// The top results must match a normal sim of the same gear, seed and iterations, and the
	// gain must be the difference from the equipped gear.
	for rank, r := range res.Results[:3] {
		single := goproto.Clone(baseSettings).(*proto.RaidSimRequest)
		for _, added := range r.ItemsAdded {
			single.Raid.Parties[0].Players[0].Equipment.Items[added.Slot] = added.Item
		}
		got := core.RunRaidSim(single).RaidMetrics.Dps.Avg
		if math.Abs(got-r.UnitMetrics.Dps.Avg) > 1e-6 {
			t.Errorf("result %d: Top Gear says %.3f DPS, a normal sim of that gear says %.3f", rank, r.UnitMetrics.Dps.Avg, got)
		}
		wantDelta := r.UnitMetrics.Dps.Avg - equipped.UnitMetrics.Dps.Avg
		if math.Abs(r.DpsDelta-wantDelta) > 1e-6 {
			t.Errorf("result %d: gain %.3f, want %.3f", rank, r.DpsDelta, wantDelta)
		}
		// Same seed for both, so comparing run by run should be less noisy than treating the
		// two sims as independent.
		independent := math.Sqrt((r.UnitMetrics.Dps.Stdev*r.UnitMetrics.Dps.Stdev + equipped.UnitMetrics.Dps.Stdev*equipped.UnitMetrics.Dps.Stdev) / iterations)
		if r.DpsDeltaStderr <= 0 || r.DpsDeltaStderr >= independent {
			t.Errorf("result %d: gain error %.3f, want between 0 and the independent error %.3f", rank, r.DpsDeltaStderr, independent)
		}
		t.Logf("#%d %+.2f ± %.2f DPS (independent error would be ± %.2f)", rank+1, r.DpsDelta, r.DpsDeltaStderr, independent)
	}

	// Fast mode must not throw away the winner: its best gain should be within noise of the full run's.
	fast := core.RunBulkSim(topGearRequest(t, iterations, true))
	if fast.Error != nil {
		t.Fatal(fast.Error.Message)
	}
	best, fastBest := res.Results[0], fast.Results[0]
	if fastBest.DpsDelta < best.DpsDelta-3*math.Hypot(best.DpsDeltaStderr, fastBest.DpsDeltaStderr) {
		t.Errorf("fast mode best gain %.2f, full run best %.2f ± %.2f", fastBest.DpsDelta, best.DpsDelta, best.DpsDeltaStderr)
	}
	t.Logf("full run best %+.2f, fast mode best %+.2f", best.DpsDelta, fastBest.DpsDelta)
}
