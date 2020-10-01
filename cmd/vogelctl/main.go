package main

import (
	"errors"
	"io"

	"github.com/andrebq/vogelnest/internal/storage"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	cmd := cobra.Command{
		Use: "vogelctl",
	}
	cmd.AddCommand(streamJSON())

	err := cmd.Execute()
	if err != nil {
		log.Error().Err(err).Send()
	}
}

func streamJSON() *cobra.Command {
	return &cobra.Command{
		Use:   "stream-json",
		Short: "Read tweets written by TweetStorageLog and write them to stdout as JSON lines",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("requires at least one file to read")
			}

			stdout := cmd.OutOrStdout()
			for _, f := range args {
				tlr, err := storage.OpenLog(f)
				if err != nil {
					return err
				}
				for {
					e, err := tlr.Next()
					if err != nil {
						break
					}
					buf, _ := protojson.Marshal(e)
					stdout.Write(buf)
					io.WriteString(stdout, "\n")
				}
			}
			_ = stdout
			return nil
		},
	}
}
