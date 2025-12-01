package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ifeitao/hugo-revise/internal/config"
	"github.com/ifeitao/hugo-revise/internal/revise"
	"github.com/ifeitao/hugo-revise/internal/undo"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "hugo-revise [PATH_PREFIX]",
		Short: "Versioned revision workflow for Hugo content",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			cfgPath, _ := cmd.Flags().GetString("config")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			return revise.Run(cfg, args[0])
		},
	}

	root.PersistentFlags().StringP("config", "c", ".hugo-reviserc.toml", "Path to config file")

	reviseCmd := &cobra.Command{
		Use:   "revise [PATH_PREFIX]",
		Short: "Create a new revision for content",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			return revise.Run(cfg, args[0])
		},
	}

	undoCmd := &cobra.Command{
		Use:   "undo",
		Short: "Undo last reviser operation",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, _ := cmd.Flags().GetString("config")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			return undo.Run(cfg)
		},
	}

	root.AddCommand(reviseCmd)
	root.AddCommand(undoCmd)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		log.Fatal(err)
	}
}
