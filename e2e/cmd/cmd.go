package cmd

import (
	"log/slog"

	"github.com/omni-network/omni/e2e/app"
	"github.com/omni-network/omni/e2e/app/agent"
	"github.com/omni-network/omni/e2e/types"
	libcmd "github.com/omni-network/omni/lib/cmd"
	"github.com/omni-network/omni/lib/log"

	cmtdocker "github.com/cometbft/cometbft/test/e2e/pkg/infra/docker"

	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	// E2E app is aimed at devs and CI, so debug level and force colors by default.
	logCfg := log.DefaultConfig()
	logCfg.Level = slog.LevelDebug.String()
	logCfg.Color = log.ColorForce

	defCfg := app.DefaultDefinitionConfig()

	var def app.Definition
	var secrets agent.Secrets

	cmd := libcmd.NewRootCmd("e2e", "e2e network generator and test runner")
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if _, err := log.Init(cmd.Context(), logCfg); err != nil {
			return err
		}

		if err := libcmd.LogFlags(cmd.Context(), cmd.Flags()); err != nil {
			return err
		}

		var err error
		def, err = app.MakeDefinition(defCfg)

		return err
	}

	bindDefFlags(cmd.PersistentFlags(), &defCfg)
	bindPromFlags(cmd.PersistentFlags(), &secrets)
	log.BindFlags(cmd.PersistentFlags(), &logCfg)

	// Root command runs the full E2E test.
	e2eTestCfg := app.DefaultE2ETestConfig()
	bindE2EFlags(cmd.Flags(), &e2eTestCfg)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return app.E2ETest(cmd.Context(), def, e2eTestCfg, secrets)
	}

	// Add subcommands
	cmd.AddCommand(
		newCreate3DeployCmd(&def),
		newAVSDeployCmd(&def),
		newDeployCmd(&def),
		newLogsCmd(&def),
		newCleanCmd(&def),
		newTestCmd(&def),
		newUpgradeCmd(&def),
	)

	return cmd
}

func newDeployCmd(def *app.Definition) *cobra.Command {
	cfg := app.DefaultDeployConfig()

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploys the e2e network",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, err := app.Deploy(cmd.Context(), *def, cfg)
			return err
		},
	}

	bindDeployFlags(cmd.Flags(), &cfg)

	return cmd
}

func newLogsCmd(def *app.Definition) *cobra.Command {
	return &cobra.Command{
		Use:   "logs",
		Short: "Prints the infrastructure logs (of a previously preserved network)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmtdocker.ExecComposeVerbose(cmd.Context(), def.Testnet.Dir, "logs")
		},
	}
}

func newCleanCmd(def *app.Definition) *cobra.Command {
	return &cobra.Command{
		Use:   "clean",
		Short: "Cleans (deletes) previously preserved network infrastructure",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return app.Cleanup(cmd.Context(), *def)
		},
	}
}

func newTestCmd(def *app.Definition) *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Runs go tests against the a previously preserved network",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return app.Test(cmd.Context(), *def, types.DeployInfos{}, true)
		},
	}
}

func newUpgradeCmd(def *app.Definition) *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrades docker containers of a previously preserved network",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return app.Upgrade(cmd.Context(), *def)
		},
	}
}

func newAVSDeployCmd(def *app.Definition) *cobra.Command {
	cfg := app.DefaultAVSDeployConfig()

	cmd := &cobra.Command{
		Use:   "avs-deploy",
		Short: "Deploys the Omni AVS contracts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.AVSDeploy(cmd.Context(), *def, cfg)
		},
	}

	bindAVSDeployFlags(cmd.Flags(), &cfg)

	return cmd
}

func newCreate3DeployCmd(def *app.Definition) *cobra.Command {
	cfg := app.DefaultCreate3DeployConfig()

	cmd := &cobra.Command{
		Use:   "create3-deploy",
		Short: "Deploys the Create3 factory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Create3Deploy(cmd.Context(), *def, cfg)
		},
	}

	bindCreate3DeployFlags(cmd.Flags(), &cfg)

	return cmd
}