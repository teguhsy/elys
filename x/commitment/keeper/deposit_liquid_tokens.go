package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	aptypes "github.com/elys-network/elys/x/assetprofile/types"
	"github.com/elys-network/elys/x/commitment/types"
)

// accounting the liquid token as a uncommitted token in commitment module.
func (k Keeper) DepositLiquidTokensUncommitted(ctx sdk.Context, denom string, amount sdk.Int, creator string) error {
	assetProfile, found := k.apKeeper.GetEntry(ctx, denom)
	if !found {
		return sdkerrors.Wrapf(aptypes.ErrAssetProfileNotFound, "denom: %s", denom)
	}

	if !assetProfile.CommitEnabled {
		return sdkerrors.Wrapf(types.ErrCommitDisabled, "denom: %s", denom)
	}

	depositCoins := sdk.NewCoins(sdk.NewCoin(denom, amount))

	addr, err := sdk.AccAddressFromBech32(creator)
	if err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, "unable to convert address from bech32")
	}

	// send the deposited coins to the module
	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, addr, types.ModuleName, depositCoins)
	if err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInsufficientFunds, fmt.Sprintf("unable to send deposit tokens: %v", depositCoins))
	}
	// burn the deposited coins
	err = k.bankKeeper.BurnCoins(ctx, types.ModuleName, depositCoins)
	if err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInsufficientFunds, "unable to burn deposit tokens")
	}

	// Get the Commitments for the creator
	commitments, found := k.GetCommitments(ctx, creator)
	if !found {
		commitments = types.Commitments{
			Creator:           creator,
			CommittedTokens:   []*types.CommittedTokens{},
			UncommittedTokens: []*types.UncommittedTokens{},
		}
	}
	// Get the uncommitted tokens for the creator
	uncommittedToken, found := commitments.GetUncommittedTokensForDenom(denom)
	if !found {
		uncommittedTokens := commitments.GetUncommittedTokens()
		uncommittedToken = &types.UncommittedTokens{
			Denom:  denom,
			Amount: sdk.ZeroInt(),
		}
		uncommittedTokens = append(uncommittedTokens, uncommittedToken)
		commitments.UncommittedTokens = uncommittedTokens
	}

	// Update the uncommitted tokens amount
	uncommittedToken.Amount = uncommittedToken.Amount.Add(amount)

	// Update the commitments
	k.SetCommitments(ctx, commitments)

	return nil
}