package buildkit

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

var remoteDriverName = "buildkitd"
var remoteDriverSocketPath = "/run/buildkit/buildkitd.sock"

// Connect the buildx to the buildkitd instance
// running on the socket (in the sidecar within this pod).
// Here buildx gets configured to use the remote driver.
// This method will block until it either see the remote driver fully
// connected or returns an unhandled error.
func ConnectRemoteDriver(ctx context.Context) error {
	logger := log.FromContext(ctx)

	cmd := CommandExecutor(ctx, "buildx", "create")
	cmd.Args = append(cmd.Args, "--driver", "remote")
	cmd.Args = append(cmd.Args, "--name", remoteDriverName)
	cmd.Args = append(cmd.Args, "--use")
	cmd.Args = append(cmd.Args, fmt.Sprintf("unix://%s", remoteDriverSocketPath))

	if err := cmd.Run(); err != nil {
		logger.Error(err, "buildx create failed")
		return err
	}

	return nil
}
