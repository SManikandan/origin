package host

import (
	"fmt"

	"github.com/openshift/origin/pkg/oc/bootstrap/docker/dockerhelper"
	"github.com/openshift/origin/pkg/oc/bootstrap/docker/errors"
	"github.com/openshift/origin/pkg/oc/bootstrap/docker/run"
)

const (
	cmdTestNsenterMount = "nsenter --mount=/rootfs/proc/1/ns/mnt findmnt"

	ensureVolumeShareCmd = `#/bin/bash
set -x
nsenter --mount=/rootfs/proc/1/ns/mnt mkdir -p %[1]s
grep -F %[1]s /rootfs/proc/1/mountinfo || nsenter --mount=/rootfs/proc/1/ns/mnt mount -o bind %[1]s %[1]s
grep -F %[1]s /rootfs/proc/1/mountinfo | grep shared || nsenter --mount=/rootfs/proc/1/ns/mnt mount --make-shared %[1]s
`
)

// HostHelper contains methods to help check settings on a Docker host machine
// using a privileged container
type HostHelper struct {
	client     dockerhelper.Interface
	runHelper  *run.RunHelper
	image      string
	volumesDir string
}

// NewHostHelper creates a new HostHelper
func NewHostHelper(dockerHelper *dockerhelper.Helper, image, volumesDir string) *HostHelper {
	return &HostHelper{
		runHelper:  run.NewRunHelper(dockerHelper),
		client:     dockerHelper.Client(),
		image:      image,
		volumesDir: volumesDir,
	}
}

// CanUseNsenterMounter returns true if the Docker host machine can execute findmnt through nsenter
func (h *HostHelper) CanUseNsenterMounter() (bool, error) {
	_, rc, err := h.runner().
		Image(h.image).
		DiscardContainer().
		Privileged().
		Bind("/:/rootfs:ro").
		Entrypoint("/bin/bash").
		Command("-c", cmdTestNsenterMount).Run()
	return err == nil && rc == 0, err
}

// EnsureVolumeUseShareMount ensures that the host Docker VM has a shared directory that can be used
// for OpenShift volumes. This is needed for Docker for Mac.
func (h *HostHelper) EnsureVolumeUseShareMount() error {
	cmd := fmt.Sprintf(ensureVolumeShareCmd, h.volumesDir)
	_, rc, err := h.runner().
		Image(h.image).
		DiscardContainer().
		HostPid().
		Privileged().
		Bind("/proc:/rootfs/proc:ro").
		Entrypoint("/bin/bash").
		Command("-c", cmd).Run()
	if err != nil || rc != 0 {
		return errors.NewError("cannot create volume share").WithCause(err)
	}
	return nil
}

func (h *HostHelper) runner() *run.Runner {
	return h.runHelper.New()
}
