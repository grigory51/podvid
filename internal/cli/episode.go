package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/grigory51/podvid/internal/config"
	"github.com/grigory51/podvid/internal/downloader"
)

func init() {
	episodeCmd := &cobra.Command{
		Use:   "episode",
		Short: "Manage podcast episodes",
	}

	episodeCmd.AddCommand(episodeAddCmd())
	episodeCmd.AddCommand(episodeListCmd())
	episodeCmd.AddCommand(episodeEditCmd())
	episodeCmd.AddCommand(episodeDeleteCmd())
	rootCmd.AddCommand(episodeCmd)
}

func episodeAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <podcast-slug> <url>",
		Short: "Download a video (any yt-dlp supported URL) and add as podcast episode",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			slug, videoURL := args[0], args[1]

			if err := downloader.EnsureFFmpeg(); err != nil {
				return err
			}

			ytdlpPath, err := downloader.EnsureYtDlp(func(s string) {
				fmt.Println(s)
			})
			if err != nil {
				return err
			}

			svc, err := newPodcastService()
			if err != nil {
				return err
			}

			cfg, _ := config.Load(cfgPath)
			dl := downloader.New(ytdlpPath, cfg.Audio.Bitrate)

			result, err := svc.AddEpisode(context.Background(), slug, videoURL, dl, func(s string) {
				fmt.Println(s)
			})
			if err != nil {
				return err
			}

			fmt.Printf("\nEpisode added: %s\n", result.Episode.Title)
			fmt.Printf("Duration: %s\n", result.Episode.Duration)
			fmt.Printf("Audio URL: %s\n", result.Episode.AudioURL)
			fmt.Printf("Feed URL: %s\n", result.FeedURL)
			return nil
		},
	}
}

func episodeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <podcast-slug>",
		Short: "List episodes of a podcast",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newPodcastService()
			if err != nil {
				return err
			}

			episodes, err := svc.ListEpisodes(context.Background(), args[0])
			if err != nil {
				return err
			}

			if len(episodes) == 0 {
				fmt.Println("No episodes found.")
				return nil
			}

			for _, ep := range episodes {
				fmt.Printf("%-12s %-40s %s  %s\n", ep.ID, ep.Title, ep.PubDate, ep.Duration)
			}
			return nil
		},
	}
}

func episodeEditCmd() *cobra.Command {
	var title, description string

	cmd := &cobra.Command{
		Use:   "edit <podcast-slug> <episode-id>",
		Short: "Edit an episode",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newPodcastService()
			if err != nil {
				return err
			}

			var titlePtr, descPtr *string
			if cmd.Flags().Changed("title") {
				titlePtr = &title
			}
			if cmd.Flags().Changed("description") {
				descPtr = &description
			}

			if err := svc.EditEpisode(context.Background(), args[0], args[1], titlePtr, descPtr); err != nil {
				return err
			}

			fmt.Println("Episode updated.")
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "new episode title")
	cmd.Flags().StringVar(&description, "description", "", "new description")
	return cmd
}

func episodeDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <podcast-slug> <episode-id>",
		Short: "Delete an episode",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newPodcastService()
			if err != nil {
				return err
			}

			if err := svc.DeleteEpisode(context.Background(), args[0], args[1]); err != nil {
				return err
			}

			fmt.Printf("Episode %q deleted from %q.\n", args[1], args[0])
			return nil
		},
	}
}
