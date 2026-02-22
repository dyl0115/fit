package cmd

import (
	"fmt"
	"os"

	"github.com/dyl01/fit/cmd/music"
	"github.com/dyl01/fit/cmd/radio"
	"github.com/dyl01/fit/cmd/youtube"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fit",
	Short: "fit - 만물 잡동사니 CLI",
	Long:  `fit은 음악 재생, 라디오 스트리밍 등 다양한 기능을 제공하는 CLI 도구입니다.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(music.MusicCmd)
	rootCmd.AddCommand(radio.RadioCmd)
	rootCmd.AddCommand(youtube.YoutubeCmd)
}
