package runtime

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/runatlantis/atlantis/server/events/models"
)

// ApplyStepRunner runs `terraform apply`.
type ApplyStepRunner struct {
	TerraformExecutor TerraformExec
}

func (a *ApplyStepRunner) Run(ctx models.ProjectCommandContext, extraArgs []string, path string) (string, error) {
	if a.hasTargetFlag(ctx, extraArgs) {
		return "", errors.New("cannot run apply with -target because we are applying an already generated plan. Instead, run -target with atlantis plan")
	}

	planPath := filepath.Join(path, GetPlanFilename(ctx.Workspace, ctx.ProjectConfig))
	contents, err := ioutil.ReadFile(planPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("no plan found at path %q and workspace %q–did you run plan?", ctx.RepoRelDir, ctx.Workspace)
	}
	if err != nil {
		return "", errors.Wrap(err, "unable to read planfile")
	}

	var tfApplyCmd []string
	if a.isRemotePlan(contents) {
		// todo: diff output during apply with output in planfile.
		tfApplyCmd = append(append(append([]string{"apply", "-input=false", "-no-color", "-auto-approve"}, extraArgs...), ctx.CommentArgs...))
	} else {
		// NOTE: we need to quote the plan path because Bitbucket Server can
		// have spaces in its repo owner names which is part of the path.
		tfApplyCmd = append(append(append([]string{"apply", "-input=false", "-no-color"}, extraArgs...), ctx.CommentArgs...), fmt.Sprintf("%q", planPath))
	}

	var tfVersion *version.Version
	if ctx.ProjectConfig != nil && ctx.ProjectConfig.TerraformVersion != nil {
		tfVersion = ctx.ProjectConfig.TerraformVersion
	}
	out, tfErr := a.TerraformExecutor.RunCommandWithVersion(ctx.Log, path, tfApplyCmd, tfVersion, ctx.Workspace)

	// If the apply was successful, delete the plan.
	if tfErr == nil {
		ctx.Log.Info("apply successful, deleting planfile")
		if err := os.Remove(planPath); err != nil {
			ctx.Log.Warn("failed to delete planfile after successful apply: %s", err)
		}
	}
	return out, tfErr
}

// isRemotePlan returns true if planContents are from a plan that was generated
// using TFE remote operations.
func (a *ApplyStepRunner) isRemotePlan(planContents []byte) bool {
	// We add a header to plans generated by the remote backend so we can
	// detect that they're remote in the apply phase.
	remoteOpsHeaderBytes := []byte(remoteOpsHeader)
	return bytes.Equal(planContents[:len(remoteOpsHeaderBytes)], remoteOpsHeaderBytes)
}

func (a *ApplyStepRunner) hasTargetFlag(ctx models.ProjectCommandContext, extraArgs []string) bool {
	isTargetFlag := func(s string) bool {
		if s == "-target" {
			return true
		}
		split := strings.Split(s, "=")
		return split[0] == "-target"
	}

	for _, arg := range ctx.CommentArgs {
		if isTargetFlag(arg) {
			return true
		}
	}
	for _, arg := range extraArgs {
		if isTargetFlag(arg) {
			return true
		}
	}
	return false
}
