package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/dockerutil"
)

// RegisterICA will attempt to register an interchain account on the counterparty chain.
func RegisterICA(t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyName, connectionID string,
) (string, error) {
	tn := chain.GetNode()
	return tn.ExecTx(ctx, keyName,
		"interchain-accounts",
		"controller",
		"register",
		connectionID,
		"--version", "",
		"--gas", "auto",
	)
}

func ExecuteICA(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyName, connectionID string,
	msg []byte,
) (string, error) {
	tn := chain.GetNode()

	file := "msg.json"
	fw := dockerutil.NewFileWriter(nil, tn.DockerClient, tn.TestName)
	if err := fw.WriteFile(ctx, tn.VolumeName, file, msg); err != nil {
		return "", fmt.Errorf("writing contract file to docker volume: %w", err)
	}

	command := []string{
		"interchain-accounts",
		"controller",
		"send-tx",
		connectionID,
		path.Join(tn.HomeDir(), file),
		"--gas", "auto",
	}

	return tn.ExecTx(ctx, keyName, command...)
}

// QueryParam returns the state and details of a subspace param.
func QueryInterchainAccount(t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	owner string,
	connectionID string,
) (string, error) {
	tn := chain.GetNode()
	stdout, _, err := tn.ExecQuery(
		ctx,
		"interchain-accounts",
		"controller",
		"interchain-account",
		owner,
		connectionID,
	)
	if err != nil {
		return "", err
	}

	var icaAccount IcaAccount
	err = json.Unmarshal(stdout, &icaAccount)
	if err != nil {
		return "", err
	}

	return icaAccount.Address, nil
}
