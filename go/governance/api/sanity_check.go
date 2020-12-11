package api

import (
	"fmt"
	"sort"

	"github.com/oasisprotocol/oasis-core/go/common/quantity"
	epochtime "github.com/oasisprotocol/oasis-core/go/epochtime/api"
	upgrade "github.com/oasisprotocol/oasis-core/go/upgrade/api"
)

// SanityCheck performs a sanity check on the consensus parameters.
func (p *ConsensusParameters) SanityCheck() error {
	if !p.MinProposalDeposit.IsValid() {
		return fmt.Errorf("min_proposal_deposit has invalid value")
	}
	// Quorum must be less or equal to 100.
	if int64(p.Quorum) > 100 {
		return fmt.Errorf("quorum should be less or equal to 100")
	}
	// Threshold must be less or equal to 100.
	if int64(p.Threshold) > 100 {
		return fmt.Errorf("threshold should be less or equal to 100")
	}
	// Quorum * threshold should be > 2/3
	if (uint64(p.Quorum) * uint64(p.Threshold)) <= 6666 {
		return fmt.Errorf("product of quorum and threshold should be at least 2/3")
	}
	// Voting_period must be less than upgrade_min_epoch_diff.
	if p.VotingPeriod >= p.UpgradeMinEpochDiff {
		return fmt.Errorf("voting_period should be less than upgrade_min_epoch_diff")
	}
	// Voting period must be less than upgrade_cancel_min_epoch_diff.
	if p.VotingPeriod >= p.UpgradeCancelMinEpochDiff {
		return fmt.Errorf("voting_period should be less than upgrade_cancel_min_epoch_diff")
	}
	return nil
}

// SanityCheckProposals sanity checks proposals.
func SanityCheckProposals(proposals []*Proposal, epoch epochtime.EpochTime, governanceDeposit *quantity.Quantity) error {
	activeProposalDeposits := quantity.NewFromUint64(0)
	for _, p := range proposals {
		if p.CreatedAt > epoch {
			return fmt.Errorf("proposal %v: with crated epoch in the future", p.ID)
		}
		if !p.Submitter.IsValid() {
			return fmt.Errorf("proposal %v: invalid proposal submitter", p.ID)
		}
		if err := p.Content.ValidateBasic(); err != nil {
			return fmt.Errorf("proposal %v: basic validation failure: %w", p.ID, err)
		}

		// XXX: There are actually other possible error states that are not covered here.
		// e.g. for cancel upgrade proposal a pending upgrade should exist.
		// These cases are not handled as in worst-case an invalid cancel proposal will result
		// in a failed proposal execution.

		switch p.State {
		case StateActive:
			if p.ClosesAt < epoch {
				return fmt.Errorf("proposal %v: active proposal with past closing epoch", p.ID)
			}
			if p.Results != nil {
				return fmt.Errorf("proposal %v: active proposal with results", p.ID)
			}
			if p.InvalidVotes != 0 {
				return fmt.Errorf("proposal %v: active proposal with non-zero invalid votes", p.ID)
			}
			if p.Content.Upgrade != nil && p.Content.Upgrade.Epoch < epoch {
				return fmt.Errorf("proposal %v: active proposal with past upgrade epoch", p.ID)
			}
			if err := activeProposalDeposits.Add(&p.Deposit); err != nil {
				return fmt.Errorf("activeProposalDeposits.Add(Deposit): %w", err)
			}

		default:
			if p.ClosesAt > epoch {
				return fmt.Errorf("proposal %v: closed proposal with future closing epoch", p.ID)
			}
		}
	}
	// Ensure active proposal deposits matches governance deposit state.
	if activeProposalDeposits.Cmp(governanceDeposit) != 0 {
		return fmt.Errorf("sum of active proposals deposits (%s) doesn't match governance deposit (%s)",
			activeProposalDeposits.String(), governanceDeposit.String(),
		)
	}
	return nil
}

// SanityCheckVotes sanity checks votes for a proposal.
func SanityCheckVotes(proposal *Proposal, votes []*VoteEntry) error {
	for _, v := range votes {
		if !v.Voter.IsValid() {
			return fmt.Errorf("proposal %v: invalid voter", proposal.ID)
		}
	}
	return nil
}

// SanityCheckPendingUpgrades sanity checks pending upgrades.
func SanityCheckPendingUpgrades(upgrades []*upgrade.Descriptor, epoch epochtime.EpochTime, params *ConsensusParameters) error {
	var upgradeEpochs []epochtime.EpochTime
	for _, up := range upgrades {
		if err := up.ValidateBasic(); err != nil {
			return fmt.Errorf("pending upgrade %v: descriptor validation failure: %w", up.Name, err)
		}
		if up.Epoch < epoch {
			return fmt.Errorf("pending upgrade %v: past upgrade epoch", up.Name)
		}
		upgradeEpochs = append(upgradeEpochs, up.Epoch)
	}
	// Ensure upgrades are UpgradeMinEpochDiff epochs apart.
	if len(upgradeEpochs) < 2 {
		return nil
	}
	sort.Slice(upgradeEpochs, func(i, j int) bool {
		return upgradeEpochs[i] < upgradeEpochs[j]
	})
	prevEpoch := upgradeEpochs[0]
	for i := 1; i < len(upgradeEpochs); i++ {
		currEpoch := upgradeEpochs[i]
		if prevEpoch.Diff(currEpoch) < params.UpgradeMinEpochDiff {
			return fmt.Errorf("pending upgrades not UpgradeMinEpochDiff(%v) apart: %v, %v",
				params.UpgradeMinEpochDiff, prevEpoch, currEpoch)
		}
		prevEpoch = currEpoch
	}

	return nil
}

// SanityCheck does basic sanity checking on the genesis state.
func (g *Genesis) SanityCheck(now epochtime.EpochTime, governanceDeposits *quantity.Quantity) error {
	if err := g.Parameters.SanityCheck(); err != nil {
		return fmt.Errorf("governance: parameters sanity check failed: %w", err)
	}
	if err := SanityCheckProposals(g.Proposals, now, governanceDeposits); err != nil {
		return fmt.Errorf("governance: proposals sanity check failed: %w", err)
	}
	for _, p := range g.Proposals {
		if err := SanityCheckVotes(p, g.VoteEntries[p.ID]); err != nil {
			return fmt.Errorf("governance: votes sanity check failed: %w", err)
		}
	}
	upgrades, _ := PendingUpgradesFromProposals(g.Proposals, now)
	if err := SanityCheckPendingUpgrades(upgrades, now, &g.Parameters); err != nil {
		return fmt.Errorf("governance: pending upgrades sanity check failed: %w", err)
	}
	return nil
}
