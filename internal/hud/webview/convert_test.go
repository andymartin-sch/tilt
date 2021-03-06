package webview

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/windmilleng/tilt/internal/engine/configs"
	"github.com/windmilleng/tilt/internal/k8s"
	"github.com/windmilleng/tilt/internal/k8s/testyaml"
	"github.com/windmilleng/tilt/internal/store"
	"github.com/windmilleng/tilt/pkg/logger"
	"github.com/windmilleng/tilt/pkg/model"
	proto_webview "github.com/windmilleng/tilt/pkg/webview"
)

var fooManifest = model.Manifest{Name: "foo"}.WithDeployTarget(model.K8sTarget{})

func stateToProtoView(t *testing.T, s store.EngineState) *proto_webview.View {
	v, err := StateToProtoView(s, 0)
	if err != nil {
		t.Fatal(err)
	}

	return v
}

func TestStateToWebViewRelativeEditPaths(t *testing.T) {
	m := model.Manifest{
		Name: "foo",
	}.WithDeployTarget(model.K8sTarget{}).WithImageTarget(model.ImageTarget{}.
		WithBuildDetails(model.DockerBuild{BuildPath: "/a/b/c"}))

	state := newState([]model.Manifest{m})
	ms := state.ManifestTargets[m.Name].State
	ms.CurrentBuild.Edits = []string{"/a/b/c/foo", "/a/b/c/d/e"}
	ms.BuildHistory = []model.BuildRecord{
		{Edits: []string{"/a/b/c/foo", "/a/b/c/d/e"}},
	}
	ms.MutableBuildStatus(m.ImageTargets[0].ID()).PendingFileChanges =
		map[string]time.Time{"/a/b/c/foo": time.Now(), "/a/b/c/d/e": time.Now()}
	v := stateToProtoView(t, *state)

	require.Len(t, v.Resources, 2)

	r, _ := findResource(m.Name, v)
	assert.ElementsMatch(t, []string{"foo", "d/e"}, lastBuild(r).Edits)

	sort.Strings(r.CurrentBuild.Edits)
	assert.ElementsMatch(t, []string{"foo", "d/e"}, r.CurrentBuild.Edits)
	assert.ElementsMatch(t, []string{"foo", "d/e"}, r.PendingBuildEdits)
}

func TestStateToWebViewPortForwards(t *testing.T) {
	m := model.Manifest{
		Name: "foo",
	}.WithDeployTarget(model.K8sTarget{
		PortForwards: []model.PortForward{
			{LocalPort: 8000, ContainerPort: 5000},
			{LocalPort: 7000, ContainerPort: 5001},
		},
	})
	state := newState([]model.Manifest{m})
	v := stateToProtoView(t, *state)
	res, _ := findResource(m.Name, v)
	assert.Equal(t,
		[]string{"http://localhost:7000/", "http://localhost:8000/"},
		res.Endpoints)
}

func TestStateToViewUnresourcedYAMLManifest(t *testing.T) {
	m, err := k8s.NewK8sOnlyManifestFromYAML(testyaml.SanchoYAML)
	assert.NoError(t, err)
	state := newState([]model.Manifest{m})
	v := stateToProtoView(t, *state)

	assert.Equal(t, 2, len(v.Resources))

	r, _ := findResource(m.Name, v)
	assert.Equal(t, "", lastBuild(r).Error)

	expectedInfo := &proto_webview.YAMLResourceInfo{K8SResources: []string{"sancho:deployment"}}
	assert.Equal(t, expectedInfo, r.YamlResourceInfo)
}

func TestStateToViewTiltfileLog(t *testing.T) {
	es := newState([]model.Manifest{})
	spanID := configs.SpanIDForLoadCount(1)
	es.LogStore.Append(
		store.NewLogAction(store.TiltfileManifestName, spanID, logger.InfoLvl, nil, []byte("hello")),
		nil)
	v := stateToProtoView(t, *es)
	r, ok := findResource("(Tiltfile)", v)
	require.True(t, ok, "no resource named (Tiltfile) found")
	assert.Equal(t, "hello", string(v.LogList.Segments[0].Text))
	assert.Equal(t, r.Name, string(v.LogList.Spans[string(spanID)].ManifestName))
}

func TestRelativeTiltfilePath(t *testing.T) {
	es := newState([]model.Manifest{})
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	es.TiltfilePath = filepath.Join(wd, "Tiltfile")

	actual, err := es.RelativeTiltfilePath()
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "Tiltfile", actual)
}

func TestNeedsNudgeSet(t *testing.T) {
	state := newState(nil)

	m := fooManifest
	targ := store.NewManifestTarget(m)
	targ.State = &store.ManifestState{}
	state.UpsertManifestTarget(targ)

	v := stateToProtoView(t, *state)

	assert.False(t, v.NeedsAnalyticsNudge,
		"LastSuccessfulDeployTime not set, so NeedsNudge should not be set")

	targ.State = &store.ManifestState{LastSuccessfulDeployTime: time.Now()}
	state.UpsertManifestTarget(targ)

	v = stateToProtoView(t, *state)
	assert.True(t, v.NeedsAnalyticsNudge)
}

func TestTriggerMode(t *testing.T) {
	state := newState(nil)
	m := fooManifest
	targ := store.NewManifestTarget(m)
	targ.Manifest.TriggerMode = model.TriggerModeManualAfterInitial
	targ.State = &store.ManifestState{}
	state.UpsertManifestTarget(targ)

	v := stateToProtoView(t, *state)
	assert.Equal(t, 2, len(v.Resources))

	newM, _ := findResource(model.ManifestName("foo"), v)
	assert.Equal(t, model.TriggerModeManualAfterInitial, model.TriggerMode(newM.TriggerMode))
}

func TestFeatureFlags(t *testing.T) {
	state := newState(nil)
	state.Features = map[string]bool{"foo_feature": true}

	v := stateToProtoView(t, *state)
	assert.Equal(t, v.FeatureFlags, map[string]bool{"foo_feature": true})
}

func TestReadinessCheckFailing(t *testing.T) {
	m := model.Manifest{
		Name: "foo",
	}.WithDeployTarget(model.K8sTarget{})
	state := newState([]model.Manifest{m})
	state.ManifestTargets[m.Name].State.RuntimeState = store.K8sRuntimeState{
		Pods: map[k8s.PodID]*store.Pod{
			"pod id": {
				Status: "Running",
				Phase:  "Running",
				Containers: []store.Container{
					{
						Ready: false,
					},
				},
			},
		},
	}

	v := stateToProtoView(t, *state)
	rv, ok := findResource(m.Name, v)
	require.True(t, ok)
	require.Equal(t, model.RuntimeStatusPending, model.RuntimeStatus(rv.RuntimeStatus))
}

func TestLocalResource(t *testing.T) {
	cmd := model.Cmd{
		Argv: []string{"make", "test"},
	}
	lt := model.NewLocalTarget("my-local", cmd, model.Cmd{}, []string{"/foo/bar", "/baz/qux"}, "path/to/tiltfile")
	m := model.Manifest{
		Name: "test",
	}.WithDeployTarget(lt)

	state := newState([]model.Manifest{m})
	state.ManifestTargets[m.Name].State.RuntimeState = store.LocalRuntimeState{
		Status: model.RuntimeStatusNotApplicable,
	}
	v := stateToProtoView(t, *state)

	assert.Equal(t, 2, len(v.Resources))
	r := v.Resources[1]
	assert.Equal(t, "test", r.Name)
	assert.Equal(t, model.RuntimeStatusNotApplicable, model.RuntimeStatus(r.RuntimeStatus))
}

func findResource(n model.ManifestName, view *proto_webview.View) (*proto_webview.Resource, bool) {
	for _, res := range view.Resources {
		if res.Name == n.String() {
			return res, true
		}
	}

	return nil, false
}

func lastBuild(r *proto_webview.Resource) *proto_webview.BuildRecord {
	if len(r.BuildHistory) == 0 {
		return &proto_webview.BuildRecord{}
	}

	return r.BuildHistory[0]
}

func newState(manifests []model.Manifest) *store.EngineState {
	ret := store.NewState()
	for _, m := range manifests {
		ret.ManifestTargets[m.Name] = store.NewManifestTarget(m)
		ret.ManifestDefinitionOrder = append(ret.ManifestDefinitionOrder, m.Name)
	}

	return ret
}
