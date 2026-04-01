package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/grigory51/podvid/internal/config"
	"github.com/grigory51/podvid/internal/tui"
)

var cfgPath string

var rootCmd = &cobra.Command{
	Use:   "podvid",
	Short: "Video to Podcast converter",
	Long:  "podvid converts videos from VK Video and RuTube into podcast episodes, uploads to S3, and manages RSS feeds.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		return tui.Run(cfg, cfgPath)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "config file path (default ~/.config/podvid/config.yaml)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
