package cli

import (
	"context"

	"github.com/kopia/kopia/internal/serverapi"
)

var (
	serverPauseCommand = serverCommands.Command("pause", "Pause the scheduled snapshots for one or more sources")
)

func init() {
	serverPauseCommand.Action(serverAction(runServerPause))
}

func runServerPause(ctx context.Context, cli *serverapi.Client) error {
	return cli.Post("sources/pause", &serverapi.Empty{}, &serverapi.Empty{})
}
