package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/grigory51/podvid/internal/config"
	"github.com/grigory51/podvid/internal/podcast"
	"github.com/grigory51/podvid/internal/storage"
)

func init() {
	podcastCmd := &cobra.Command{
		Use:   "podcast",
		Short: "Manage podcasts",
	}

	podcastCmd.AddCommand(podcastCreateCmd())
	podcastCmd.AddCommand(podcastListCmd())
	podcastCmd.AddCommand(podcastEditCmd())
	podcastCmd.AddCommand(podcastDeleteCmd())
	rootCmd.AddCommand(podcastCmd)
}

func podcastCreateCmd() *cobra.Command {
	var name, description, cover string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new podcast",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newPodcastService()
			if err != nil {
				return err
			}

			info, err := svc.Create(context.Background(), name, description)
			if err != nil {
				return err
			}

			if cover != "" {
				data, err := os.ReadFile(cover)
				if err != nil {
					return fmt.Errorf("reading cover image: %w", err)
				}
				ct := "image/jpeg"
				if len(cover) > 4 && cover[len(cover)-4:] == ".png" {
					ct = "image/png"
				}
				if err := svc.SetCover(context.Background(), info.Slug, data, ct); err != nil {
					return fmt.Errorf("uploading cover: %w", err)
				}
			}

			fmt.Printf("Podcast created: %s\n", info.Name)
			fmt.Printf("Slug: %s\n", info.Slug)
			fmt.Printf("Feed URL: %s\n", info.FeedURL)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "podcast name (required)")
	cmd.Flags().StringVar(&description, "description", "", "podcast description")
	cmd.Flags().StringVar(&cover, "cover", "", "path to cover image")
	cmd.MarkFlagRequired("name")
	return cmd
}

func podcastListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all podcasts",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newPodcastService()
			if err != nil {
				return err
			}

			podcasts, err := svc.List(context.Background())
			if err != nil {
				return err
			}

			if len(podcasts) == 0 {
				fmt.Println("No podcasts found.")
				return nil
			}

			for _, p := range podcasts {
				fmt.Printf("%-20s %d episodes  %s\n", p.Slug, p.EpisodeCount, p.FeedURL)
			}
			return nil
		},
	}
}

func podcastEditCmd() *cobra.Command {
	var name, description, cover string

	cmd := &cobra.Command{
		Use:   "edit <slug>",
		Short: "Edit a podcast",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newPodcastService()
			if err != nil {
				return err
			}
			slug := args[0]

			var namePtr, descPtr *string
			if cmd.Flags().Changed("name") {
				namePtr = &name
			}
			if cmd.Flags().Changed("description") {
				descPtr = &description
			}

			if err := svc.Edit(context.Background(), slug, namePtr, descPtr); err != nil {
				return err
			}

			if cover != "" {
				data, err := os.ReadFile(cover)
				if err != nil {
					return fmt.Errorf("reading cover image: %w", err)
				}
				ct := "image/jpeg"
				if len(cover) > 4 && cover[len(cover)-4:] == ".png" {
					ct = "image/png"
				}
				if err := svc.SetCover(context.Background(), slug, data, ct); err != nil {
					return fmt.Errorf("uploading cover: %w", err)
				}
			}

			fmt.Println("Podcast updated.")
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "new podcast name")
	cmd.Flags().StringVar(&description, "description", "", "new description")
	cmd.Flags().StringVar(&cover, "cover", "", "new cover image path")
	return cmd
}

func podcastDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a podcast and all its episodes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newPodcastService()
			if err != nil {
				return err
			}

			if err := svc.Delete(context.Background(), args[0]); err != nil {
				return err
			}

			fmt.Printf("Podcast %q deleted.\n", args[0])
			return nil
		},
	}
}

func newPodcastService() (*podcast.Service, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w\nRun 'podvid config init' to set up", err)
	}

	s3Client, err := storage.NewS3Client(cfg)
	if err != nil {
		return nil, err
	}

	return podcast.NewService(s3Client, cfg), nil
}
