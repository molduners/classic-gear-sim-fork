package cmd

import (
	"fmt"
	"log"
	"math"
	"os"
	"sort"

	"github.com/spf13/cobra"
	"github.com/wowsims/classic/sim/core"
	"github.com/wowsims/classic/sim/core/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

var simCmd = &cobra.Command{
	Use:   "sim",
	Short: "simulate items & settings",
	Run:   simMain,
}

func init() {
	simCmd.Flags().StringVar(&infile, "infile", "input.json", "location of input file (RaidSimRequest in protojson format)")
	simCmd.Flags().StringVar(&outfile, "outfile", "", "location of output file, defaults to stdout")
	simCmd.Flags().BoolVar(&verbose, "verbose", false, "print information during runtime")
	simCmd.Flags().BoolVar(&summary, "summary", false, "print a short readable summary instead of the full JSON result")
	simCmd.MarkFlagRequired("infile")
}

func simMain(cmd *cobra.Command, args []string) {
	data, err := os.ReadFile(infile)
	if err != nil {
		log.Fatalf("failed to load input json file %q: %v", infile, err)
	}
	input := &proto.RaidSimRequest{}

	err = protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(data, input)
	if err != nil {
		log.Fatalf("failed to load input json file: %s", err)
	}

	var output []byte
	reporter := make(chan *proto.ProgressMetrics, 10)
	core.RunRaidSimConcurrentAsync(input, reporter, "cmd-raid-sim")

	var finalResult *proto.RaidSimResult
	for v := range reporter {
		if v.FinalRaidResult != nil {
			finalResult = v.FinalRaidResult
			break
		}
		if verbose {
			fmt.Printf("Sim Progress: %d / %d\n", v.CompletedIterations, v.TotalIterations)
		}
	}

	if summary {
		printSummary(input, finalResult)
		return
	}

	output, err = protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(finalResult)
	if err != nil {
		log.Fatalf("failed to marshal final results: %s", err)
	}

	if outfile == "" {
		fmt.Print(string(output))
	} else {
		err = os.WriteFile(outfile, output, 0666)
		if err != nil {
			log.Fatalf("failed to write output file:: %s", err)
		}
		if verbose {
			fmt.Printf("Wrote output file: `%s` successfully.\n", outfile)
		}
	}
}

var summary bool

// printSummary prints each player's DPS (with its margin of error) and a damage breakdown by action.
func printSummary(input *proto.RaidSimRequest, result *proto.RaidSimResult) {
	if result.Error != nil {
		log.Fatalf("sim failed: %s", result.Error.Message)
	}
	iterations := float64(input.SimOptions.Iterations)
	seconds := input.Encounter.Duration
	fmt.Printf("%d iterations of a %.0f second fight\n", input.SimOptions.Iterations, seconds)
	for _, party := range result.RaidMetrics.Parties {
		for _, player := range party.Players {
			if player.Name == "" {
				continue
			}
			dps := player.Dps
			fmt.Printf("\n%s: %.1f DPS (± %.1f, range %.0f to %.0f)\n", player.Name, dps.Avg, dps.Stdev/math.Sqrt(iterations), dps.Min, dps.Max)
			actions := player.Actions
			damage := func(a *proto.ActionMetrics) float64 {
				total := 0.0
				for _, t := range a.Targets {
					total += t.Damage
				}
				return total
			}
			sort.Slice(actions, func(i, j int) bool { return damage(actions[i]) > damage(actions[j]) })
			for _, a := range actions {
				if d := damage(a) / iterations / seconds; d >= 0.05 {
					fmt.Printf("  %-40s %7.1f DPS  %5.1f%%\n", a.Id.String(), d, 100*d/dps.Avg)
				}
			}
		}
	}
}
