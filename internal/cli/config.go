package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/grigory51/podvid/internal/config"
)

func init() {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	configCmd.AddCommand(configInitCmd())
	configCmd.AddCommand(configShowCmd())
	rootCmd.AddCommand(configCmd)
}

func configInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := &config.Config{
				Audio: config.AudioConfig{Bitrate: "192k"},
			}

			reader := bufio.NewReader(os.Stdin)

			cfg.S3.Endpoint = prompt(reader, "S3 Endpoint (leave empty for AWS)", "")
			cfg.S3.Region = prompt(reader, "S3 Region", "us-east-1")
			cfg.S3.Bucket = prompt(reader, "S3 Bucket", "")
			cfg.S3.AccessKey = prompt(reader, "S3 Access Key", "")
			cfg.S3.SecretKey = prompt(reader, "S3 Secret Key", "")
			cfg.S3.PublicBaseURL = prompt(reader, "S3 Public Base URL (for RSS links)", "")
			cfg.Audio.Bitrate = prompt(reader, "Audio Bitrate", "192k")

			path := cfgPath
			if path == "" {
				path = config.DefaultConfigPath()
			}

			if err := cfg.Save(path); err != nil {
				return err
			}

			fmt.Printf("\nConfig saved to %s\n", path)
			return nil
		},
	}
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			fmt.Print(cfg.Display())
			return nil
		},
	}
}

func prompt(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("%s: ", label)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}
