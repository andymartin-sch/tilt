package engine

import (
	"context"
	"sort"
	"time"

	"github.com/windmilleng/tilt/internal/logger"
	"github.com/windmilleng/tilt/internal/model"
	"github.com/windmilleng/tilt/internal/ospath"
	"github.com/windmilleng/tilt/internal/store"
	"github.com/windmilleng/tilt/internal/tiltfile"
)

type BuildController struct {
	b               BuildAndDeployer
	lastActionCount int
}

type buildEntry struct {
	ctx               context.Context
	manifest          model.Manifest
	buildState        store.BuildState
	filesChanged      []string
	firstBuild        bool
	needsConfigReload bool
}

func NewBuildController(b BuildAndDeployer) *BuildController {
	return &BuildController{
		b:               b,
		lastActionCount: -1,
	}
}

func (c *BuildController) needsBuild(ctx context.Context, st *store.Store) (buildEntry, bool) {
	state := st.RLockState()
	defer st.RUnlockState()

	if len(state.ManifestsToBuild) == 0 {
		return buildEntry{}, false
	}

	if c.lastActionCount == state.BuildControllerActionCount {
		return buildEntry{}, false
	}

	mn := state.ManifestsToBuild[0]
	c.lastActionCount = state.BuildControllerActionCount
	ms := state.ManifestStates[mn]
	manifest := ms.Manifest
	firstBuild := !ms.HasBeenBuilt

	filesChanged := make([]string, 0, len(ms.PendingFileChanges))
	for file, _ := range ms.PendingFileChanges {
		filesChanged = append(filesChanged, file)
	}
	sort.Strings(filesChanged)

	buildState := store.NewBuildState(ms.LastBuild, filesChanged)

	needsConfigReload := ms.ConfigIsDirty

	// TODO(nick): This is...not great, because it modifies the build log in place.
	// A better solution would dispatch actions (like PodLogManager does) so that
	// they go thru the state loop and immediately update in the UX.
	ctx = logger.CtxWithForkedOutput(ctx, ms.CurrentBuildLog)

	return buildEntry{
		ctx:               ctx,
		manifest:          manifest,
		firstBuild:        firstBuild,
		buildState:        buildState,
		filesChanged:      filesChanged,
		needsConfigReload: needsConfigReload,
	}, true
}

func (c *BuildController) OnChange(ctx context.Context, st *store.Store) {
	entry, ok := c.needsBuild(ctx, st)
	if !ok {
		return
	}

	go func() {
		if entry.needsConfigReload {
			newManifest, err := getNewManifestFromTiltfile(entry.ctx, entry.manifest.Name)
			st.Dispatch(ManifestReloadedAction{
				OldManifest: entry.manifest,
				NewManifest: newManifest,
				Error:       err,
			})
			return
		}

		st.Dispatch(BuildStartedAction{
			Manifest:     entry.manifest,
			StartTime:    time.Now(),
			FilesChanged: entry.filesChanged,
		})
		c.logBuildEntry(entry.ctx, entry)
		result, err := c.b.BuildAndDeploy(entry.ctx, entry.manifest, entry.buildState)
		st.Dispatch(NewBuildCompleteAction(result, err))
	}()
}

func (c *BuildController) logBuildEntry(ctx context.Context, entry buildEntry) {
	firstBuild := entry.firstBuild
	manifest := entry.manifest
	buildState := entry.buildState

	l := logger.Get(ctx)
	if firstBuild {
		p := logger.Blue(l).Sprintf("──┤ Building: ")
		s := logger.Blue(l).Sprintf(" ├──────────────────────────────────────────────")
		l.Infof("%s%s%s", p, manifest.Name, s)
	} else {
		changedFiles := buildState.FilesChanged()
		var changedPathsToPrint []string
		if len(changedFiles) > maxChangedFilesToPrint {
			changedPathsToPrint = append(changedPathsToPrint, changedFiles[:maxChangedFilesToPrint]...)
			changedPathsToPrint = append(changedPathsToPrint, "...")
		} else {
			changedPathsToPrint = changedFiles
		}

		p := logger.Green(l).Sprintf("\n%d changed: ", len(changedFiles))
		l.Infof("%s%v\n", p, ospath.TryAsCwdChildren(changedPathsToPrint))
		rp := logger.Blue(l).Sprintf("──┤ Rebuilding: ")
		rs := logger.Blue(l).Sprintf(" ├────────────────────────────────────────────")
		l.Infof("%s%s%s", rp, manifest.Name, rs)
	}
}

func getNewManifestFromTiltfile(ctx context.Context, name model.ManifestName) (model.Manifest, *manifestErr) {
	// Sends any output to the CurrentBuildLog
	writer := logger.Get(ctx).Writer(logger.InfoLvl)
	t, err := tiltfile.Load(tiltfile.FileName, writer)
	if err != nil {
		return model.Manifest{}, manifestErrf(err.Error())
	}
	newManifests, err := t.GetManifestConfigs(string(name))
	if err != nil {
		return model.Manifest{}, manifestErrf(err.Error())
	}
	if len(newManifests) != 1 {
		return model.Manifest{}, manifestErrf("Expected there to be 1 manifest for %s, got %d", name, len(newManifests))
	}
	newManifest := newManifests[0]

	return newManifest, nil
}

var _ store.Subscriber = &BuildController{}
