package solveoutbox

import (
	"context"

	"github.com/omni-network/omni/contracts/bindings"
	"github.com/omni-network/omni/e2e/app/eoa"
	"github.com/omni-network/omni/lib/contracts"
	"github.com/omni-network/omni/lib/create3"
	"github.com/omni-network/omni/lib/errors"
	"github.com/omni-network/omni/lib/ethclient/ethbackend"
	"github.com/omni-network/omni/lib/netconf"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

type DeploymentConfig struct {
	Create3Factory  common.Address
	Create3Salt     string
	ProxyAdminOwner common.Address
	Owner           common.Address
	Solver          common.Address
	Portal          common.Address
	Inbox           common.Address
	Deployer        common.Address
	ExpectedAddr    common.Address
}

func (cfg DeploymentConfig) Validate() error {
	if (cfg.Create3Factory == common.Address{}) {
		return errors.New("create3 factory is zero")
	}
	if cfg.Create3Salt == "" {
		return errors.New("create3 salt is empty")
	}
	if (cfg.ProxyAdminOwner == common.Address{}) {
		return errors.New("proxy admin is zero")
	}
	if contracts.IsEmptyAddress(cfg.Deployer) {
		return errors.New("deployer is not set")
	}
	if contracts.IsEmptyAddress(cfg.Owner) {
		return errors.New("owner is not set")
	}
	if (cfg.Inbox == common.Address{}) {
		return errors.New("inbox is zero")
	}
	if (cfg.Portal == common.Address{}) {
		return errors.New("portal is zero")
	}
	if (cfg.Solver == common.Address{}) {
		return errors.New("solver is zero")
	}
	if (cfg.ExpectedAddr == common.Address{}) {
		return errors.New("expected address is zero")
	}

	return nil
}

// isDeployed returns true if the SolveOutbox contract is already deployed to its expected address.
func isDeployed(ctx context.Context, network netconf.ID, backend *ethbackend.Backend) (bool, common.Address, error) {
	addrs, err := contracts.GetAddresses(ctx, network)
	if err != nil {
		return false, common.Address{}, errors.Wrap(err, "get addresses")
	}

	addr := addrs.SolveOutbox

	code, err := backend.CodeAt(ctx, addr, nil)
	if err != nil {
		return false, addr, errors.Wrap(err, "code at", "address", addr)
	}

	if len(code) == 0 {
		return false, addr, nil
	}

	return true, addr, nil
}

// DeployIfNeeded deploys a new SolveOutbox contract if it is not already deployed.
// If the contract is already deployed, the receipt is nil.
func DeployIfNeeded(ctx context.Context, network netconf.ID, backend *ethbackend.Backend) (common.Address, *ethtypes.Receipt, error) {
	deployed, addr, err := isDeployed(ctx, network, backend)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "is deployed")
	}
	if deployed {
		return addr, nil, nil
	}

	return Deploy(ctx, network, backend)
}

// Deploy deploys a new SolveOutbox contract and returns the address and receipt.
func Deploy(ctx context.Context, network netconf.ID, backend *ethbackend.Backend) (common.Address, *ethtypes.Receipt, error) {
	addrs, err := contracts.GetAddresses(ctx, network)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "get addresses")
	}

	salts, err := contracts.GetSalts(ctx, network)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "get salts")
	}

	cfg := DeploymentConfig{
		Create3Factory:  addrs.Create3Factory,
		Create3Salt:     salts.SolveOutbox,
		Owner:           eoa.MustAddress(network, eoa.RoleManager),
		Deployer:        eoa.MustAddress(network, eoa.RoleDeployer),
		ProxyAdminOwner: eoa.MustAddress(network, eoa.RoleUpgrader),
		Solver:          eoa.MustAddress(network, eoa.RoleSolver),
		Portal:          addrs.Portal,
		Inbox:           addrs.SolveInbox,
		ExpectedAddr:    addrs.SolveOutbox,
	}

	return deploy(ctx, cfg, backend)
}

func deploy(ctx context.Context, cfg DeploymentConfig, backend *ethbackend.Backend) (common.Address, *ethtypes.Receipt, error) {
	if err := cfg.Validate(); err != nil {
		return common.Address{}, nil, errors.Wrap(err, "validate config")
	}

	txOpts, err := backend.BindOpts(ctx, cfg.Deployer)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "bind opts")
	}

	factory, err := bindings.NewCreate3(cfg.Create3Factory, backend)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "new create3")
	}

	salt := create3.HashSalt(cfg.Create3Salt)

	addr, err := factory.GetDeployed(nil, txOpts.From, salt)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "get deployed")
	} else if (cfg.ExpectedAddr != common.Address{}) && addr != cfg.ExpectedAddr {
		return common.Address{}, nil, errors.New("unexpected address", "expected", cfg.ExpectedAddr, "actual", addr)
	}

	impl, tx, _, err := bindings.DeploySolveOutbox(txOpts, backend)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "deploy impl")
	}

	_, err = backend.WaitMined(ctx, tx)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "wait mined impl")
	}

	initCode, err := packInitCode(cfg, impl)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "pack init code")
	}

	tx, err = factory.DeployWithRetry(txOpts, salt, initCode) //nolint:contextcheck // Context is txOpts
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "deploy proxy")
	}

	receipt, err := backend.WaitMined(ctx, tx)
	if err != nil {
		return common.Address{}, nil, errors.Wrap(err, "wait mined proxy")
	}

	return addr, receipt, nil
}

func packInitCode(cfg DeploymentConfig, impl common.Address) ([]byte, error) {
	outboxAbi, err := bindings.SolveOutboxMetaData.GetAbi()
	if err != nil {
		return nil, errors.Wrap(err, "get abi")
	}

	proxyAbi, err := bindings.TransparentUpgradeableProxyMetaData.GetAbi()
	if err != nil {
		return nil, errors.Wrap(err, "get proxy abi")
	}

	initializer, err := outboxAbi.Pack("initialize", cfg.Owner, cfg.Solver, cfg.Portal, cfg.Inbox)
	if err != nil {
		return nil, errors.Wrap(err, "encode initializer")
	}

	return contracts.PackInitCode(proxyAbi, bindings.TransparentUpgradeableProxyBin, impl, cfg.ProxyAdminOwner, initializer)
}