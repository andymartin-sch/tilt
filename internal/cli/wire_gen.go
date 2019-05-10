// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package cli

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/google/wire"
	"github.com/windmilleng/tilt/internal/assets"
	"github.com/windmilleng/tilt/internal/build"
	"github.com/windmilleng/tilt/internal/container"
	"github.com/windmilleng/tilt/internal/demo"
	"github.com/windmilleng/tilt/internal/docker"
	"github.com/windmilleng/tilt/internal/dockercompose"
	"github.com/windmilleng/tilt/internal/dockerfile"
	"github.com/windmilleng/tilt/internal/engine"
	"github.com/windmilleng/tilt/internal/hud"
	"github.com/windmilleng/tilt/internal/hud/server"
	"github.com/windmilleng/tilt/internal/k8s"
	"github.com/windmilleng/tilt/internal/minikube"
	"github.com/windmilleng/tilt/internal/model"
	"github.com/windmilleng/tilt/internal/sail/client"
	"github.com/windmilleng/tilt/internal/store"
	"github.com/windmilleng/tilt/internal/tiltfile"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/tools/clientcmd/api"
	"time"
)

// Injectors from wire.go:

func wireDemo(ctx context.Context, branch demo.RepoBranch) (demo.Script, error) {
	reducer := _wireReducerValue
	storeLogActionsFlag := provideLogActions()
	storeStore := store.NewStore(reducer, storeLogActionsFlag)
	v := provideClock()
	renderer := hud.NewRenderer(v)
	modelWebPort := provideWebPort()
	webURL, err := provideWebURL(modelWebPort)
	if err != nil {
		return demo.Script{}, err
	}
	analytics, err := provideAnalytics()
	if err != nil {
		return demo.Script{}, err
	}
	headsUpDisplay, err := hud.NewDefaultHeadsUpDisplay(renderer, webURL, analytics)
	if err != nil {
		return demo.Script{}, err
	}
	clientConfig := k8s.ProvideClientConfig()
	config, err := k8s.ProvideKubeConfig(clientConfig)
	if err != nil {
		return demo.Script{}, err
	}
	env := k8s.ProvideEnv(config)
	portForwarder := k8s.ProvidePortForwarder()
	namespace := k8s.ProvideConfigNamespace(clientConfig)
	kubeContext, err := k8s.ProvideKubeContext(config)
	if err != nil {
		return demo.Script{}, err
	}
	kubectlRunner := k8s.ProvideKubectlRunner(kubeContext)
	k8sClient := k8s.ProvideK8sClient(ctx, env, portForwarder, namespace, kubectlRunner, clientConfig)
	podWatcher := engine.NewPodWatcher(k8sClient)
	nodeIP, err := k8s.DetectNodeIP(ctx, env)
	if err != nil {
		return demo.Script{}, err
	}
	serviceWatcher := engine.NewServiceWatcher(k8sClient, nodeIP)
	podLogManager := engine.NewPodLogManager(k8sClient)
	portForwardController := engine.NewPortForwardController(k8sClient)
	fsWatcherMaker := engine.ProvideFsWatcherMaker()
	timerMaker := engine.ProvideTimerMaker()
	watchManager := engine.NewWatchManager(fsWatcherMaker, timerMaker)
	syncletManager := engine.NewSyncletManager(k8sClient)
	engineUpdateModeFlag := provideUpdateModeFlag()
	runtime := k8s.ProvideContainerRuntime(ctx, k8sClient)
	updateMode, err := engine.ProvideUpdateMode(engineUpdateModeFlag, env, runtime)
	if err != nil {
		return demo.Script{}, err
	}
	syncletBuildAndDeployer := engine.NewSyncletBuildAndDeployer(syncletManager, k8sClient, updateMode)
	minikubeClient := minikube.ProvideMinikubeClient()
	dockerEnv, err := docker.ProvideEnv(ctx, env, runtime, minikubeClient)
	if err != nil {
		return demo.Script{}, err
	}
	clientClient, err := docker.ProvideDockerClient(ctx, dockerEnv)
	if err != nil {
		return demo.Script{}, err
	}
	version, err := docker.ProvideDockerVersion(ctx, clientClient)
	if err != nil {
		return demo.Script{}, err
	}
	cli, err := docker.DefaultClient(ctx, clientClient, version)
	if err != nil {
		return demo.Script{}, err
	}
	containerUpdater := build.NewContainerUpdater(cli)
	localContainerBuildAndDeployer := engine.NewLocalContainerBuildAndDeployer(containerUpdater, analytics, env)
	labels := _wireLabelsValue
	dockerImageBuilder := build.NewDockerImageBuilder(cli, labels)
	imageBuilder := build.DefaultImageBuilder(dockerImageBuilder)
	cacheBuilder := build.NewCacheBuilder(cli)
	clock := build.ProvideClock()
	execCustomBuilder := build.NewExecCustomBuilder(cli, dockerEnv, clock)
	kindPusher := engine.NewKINDPusher()
	imageBuildAndDeployer := engine.NewImageBuildAndDeployer(imageBuilder, cacheBuilder, execCustomBuilder, k8sClient, env, analytics, updateMode, clock, runtime, kindPusher)
	dockerComposeClient := dockercompose.NewDockerComposeClient(dockerEnv)
	imageAndCacheBuilder := engine.NewImageAndCacheBuilder(imageBuilder, cacheBuilder, execCustomBuilder, updateMode)
	dockerComposeBuildAndDeployer := engine.NewDockerComposeBuildAndDeployer(dockerComposeClient, cli, imageAndCacheBuilder, clock)
	buildOrder := engine.DefaultBuildOrder(syncletBuildAndDeployer, localContainerBuildAndDeployer, imageBuildAndDeployer, dockerComposeBuildAndDeployer, env, updateMode, runtime)
	compositeBuildAndDeployer := engine.NewCompositeBuildAndDeployer(buildOrder)
	buildController := engine.NewBuildController(compositeBuildAndDeployer)
	imageReaper := build.NewImageReaper(cli)
	imageController := engine.NewImageController(imageReaper)
	tiltfileLoader := tiltfile.ProvideTiltfileLoader(analytics, dockerComposeClient)
	configsController := engine.NewConfigsController(tiltfileLoader)
	dockerComposeEventWatcher := engine.NewDockerComposeEventWatcher(dockerComposeClient)
	dockerComposeLogManager := engine.NewDockerComposeLogManager(dockerComposeClient)
	profilerManager := engine.NewProfilerManager()
	analyticsReporter := engine.ProvideAnalyticsReporter(analytics, storeStore)
	tiltBuild := provideTiltInfo()
	webMode, err := provideWebMode(tiltBuild)
	if err != nil {
		return demo.Script{}, err
	}
	webVersion := provideWebVersion(tiltBuild)
	modelWebDevPort := provideWebDevPort()
	assetsServer, err := assets.ProvideAssetServer(ctx, webMode, webVersion, modelWebDevPort)
	if err != nil {
		return demo.Script{}, err
	}
	sailMode, err := provideSailMode()
	if err != nil {
		return demo.Script{}, err
	}
	sailURL, err := provideSailURL(sailMode)
	if err != nil {
		return demo.Script{}, err
	}
	sailRoomer := client.ProvideSailRoomer(sailURL)
	sailDialer := client.ProvideSailDialer()
	sailClient := client.ProvideSailClient(sailURL, sailRoomer, sailDialer)
	headsUpServer := server.ProvideHeadsUpServer(storeStore, assetsServer, analytics, sailClient)
	headsUpServerController := server.ProvideHeadsUpServerController(modelWebPort, headsUpServer, assetsServer, webURL)
	githubClientFactory := engine.NewGithubClientFactory()
	tiltVersionChecker := engine.NewTiltVersionChecker(githubClientFactory, timerMaker)
	v2 := engine.ProvideSubscribers(headsUpDisplay, podWatcher, serviceWatcher, podLogManager, portForwardController, watchManager, buildController, imageController, configsController, dockerComposeEventWatcher, dockerComposeLogManager, profilerManager, syncletManager, analyticsReporter, headsUpServerController, sailClient, tiltVersionChecker)
	upper := engine.NewUpper(ctx, storeStore, v2)
	script := demo.NewScript(upper, headsUpDisplay, k8sClient, env, storeStore, branch, runtime, tiltfileLoader)
	return script, nil
}

var (
	_wireReducerValue = engine.UpperReducer
	_wireLabelsValue  = dockerfile.Labels{}
)

func wireThreads(ctx context.Context) (Threads, error) {
	v := provideClock()
	renderer := hud.NewRenderer(v)
	modelWebPort := provideWebPort()
	webURL, err := provideWebURL(modelWebPort)
	if err != nil {
		return Threads{}, err
	}
	analytics, err := provideAnalytics()
	if err != nil {
		return Threads{}, err
	}
	headsUpDisplay, err := hud.NewDefaultHeadsUpDisplay(renderer, webURL, analytics)
	if err != nil {
		return Threads{}, err
	}
	reducer := _wireReducerValue
	storeLogActionsFlag := provideLogActions()
	storeStore := store.NewStore(reducer, storeLogActionsFlag)
	clientConfig := k8s.ProvideClientConfig()
	config, err := k8s.ProvideKubeConfig(clientConfig)
	if err != nil {
		return Threads{}, err
	}
	env := k8s.ProvideEnv(config)
	portForwarder := k8s.ProvidePortForwarder()
	namespace := k8s.ProvideConfigNamespace(clientConfig)
	kubeContext, err := k8s.ProvideKubeContext(config)
	if err != nil {
		return Threads{}, err
	}
	kubectlRunner := k8s.ProvideKubectlRunner(kubeContext)
	k8sClient := k8s.ProvideK8sClient(ctx, env, portForwarder, namespace, kubectlRunner, clientConfig)
	podWatcher := engine.NewPodWatcher(k8sClient)
	nodeIP, err := k8s.DetectNodeIP(ctx, env)
	if err != nil {
		return Threads{}, err
	}
	serviceWatcher := engine.NewServiceWatcher(k8sClient, nodeIP)
	podLogManager := engine.NewPodLogManager(k8sClient)
	portForwardController := engine.NewPortForwardController(k8sClient)
	fsWatcherMaker := engine.ProvideFsWatcherMaker()
	timerMaker := engine.ProvideTimerMaker()
	watchManager := engine.NewWatchManager(fsWatcherMaker, timerMaker)
	syncletManager := engine.NewSyncletManager(k8sClient)
	engineUpdateModeFlag := provideUpdateModeFlag()
	runtime := k8s.ProvideContainerRuntime(ctx, k8sClient)
	updateMode, err := engine.ProvideUpdateMode(engineUpdateModeFlag, env, runtime)
	if err != nil {
		return Threads{}, err
	}
	syncletBuildAndDeployer := engine.NewSyncletBuildAndDeployer(syncletManager, k8sClient, updateMode)
	minikubeClient := minikube.ProvideMinikubeClient()
	dockerEnv, err := docker.ProvideEnv(ctx, env, runtime, minikubeClient)
	if err != nil {
		return Threads{}, err
	}
	clientClient, err := docker.ProvideDockerClient(ctx, dockerEnv)
	if err != nil {
		return Threads{}, err
	}
	version, err := docker.ProvideDockerVersion(ctx, clientClient)
	if err != nil {
		return Threads{}, err
	}
	cli, err := docker.DefaultClient(ctx, clientClient, version)
	if err != nil {
		return Threads{}, err
	}
	containerUpdater := build.NewContainerUpdater(cli)
	localContainerBuildAndDeployer := engine.NewLocalContainerBuildAndDeployer(containerUpdater, analytics, env)
	labels := _wireLabelsValue
	dockerImageBuilder := build.NewDockerImageBuilder(cli, labels)
	imageBuilder := build.DefaultImageBuilder(dockerImageBuilder)
	cacheBuilder := build.NewCacheBuilder(cli)
	clock := build.ProvideClock()
	execCustomBuilder := build.NewExecCustomBuilder(cli, dockerEnv, clock)
	kindPusher := engine.NewKINDPusher()
	imageBuildAndDeployer := engine.NewImageBuildAndDeployer(imageBuilder, cacheBuilder, execCustomBuilder, k8sClient, env, analytics, updateMode, clock, runtime, kindPusher)
	dockerComposeClient := dockercompose.NewDockerComposeClient(dockerEnv)
	imageAndCacheBuilder := engine.NewImageAndCacheBuilder(imageBuilder, cacheBuilder, execCustomBuilder, updateMode)
	dockerComposeBuildAndDeployer := engine.NewDockerComposeBuildAndDeployer(dockerComposeClient, cli, imageAndCacheBuilder, clock)
	buildOrder := engine.DefaultBuildOrder(syncletBuildAndDeployer, localContainerBuildAndDeployer, imageBuildAndDeployer, dockerComposeBuildAndDeployer, env, updateMode, runtime)
	compositeBuildAndDeployer := engine.NewCompositeBuildAndDeployer(buildOrder)
	buildController := engine.NewBuildController(compositeBuildAndDeployer)
	imageReaper := build.NewImageReaper(cli)
	imageController := engine.NewImageController(imageReaper)
	tiltfileLoader := tiltfile.ProvideTiltfileLoader(analytics, dockerComposeClient)
	configsController := engine.NewConfigsController(tiltfileLoader)
	dockerComposeEventWatcher := engine.NewDockerComposeEventWatcher(dockerComposeClient)
	dockerComposeLogManager := engine.NewDockerComposeLogManager(dockerComposeClient)
	profilerManager := engine.NewProfilerManager()
	analyticsReporter := engine.ProvideAnalyticsReporter(analytics, storeStore)
	tiltBuild := provideTiltInfo()
	webMode, err := provideWebMode(tiltBuild)
	if err != nil {
		return Threads{}, err
	}
	webVersion := provideWebVersion(tiltBuild)
	modelWebDevPort := provideWebDevPort()
	assetsServer, err := assets.ProvideAssetServer(ctx, webMode, webVersion, modelWebDevPort)
	if err != nil {
		return Threads{}, err
	}
	sailMode, err := provideSailMode()
	if err != nil {
		return Threads{}, err
	}
	sailURL, err := provideSailURL(sailMode)
	if err != nil {
		return Threads{}, err
	}
	sailRoomer := client.ProvideSailRoomer(sailURL)
	sailDialer := client.ProvideSailDialer()
	sailClient := client.ProvideSailClient(sailURL, sailRoomer, sailDialer)
	headsUpServer := server.ProvideHeadsUpServer(storeStore, assetsServer, analytics, sailClient)
	headsUpServerController := server.ProvideHeadsUpServerController(modelWebPort, headsUpServer, assetsServer, webURL)
	githubClientFactory := engine.NewGithubClientFactory()
	tiltVersionChecker := engine.NewTiltVersionChecker(githubClientFactory, timerMaker)
	v2 := engine.ProvideSubscribers(headsUpDisplay, podWatcher, serviceWatcher, podLogManager, portForwardController, watchManager, buildController, imageController, configsController, dockerComposeEventWatcher, dockerComposeLogManager, profilerManager, syncletManager, analyticsReporter, headsUpServerController, sailClient, tiltVersionChecker)
	upper := engine.NewUpper(ctx, storeStore, v2)
	threads := provideThreads(headsUpDisplay, upper, tiltBuild, sailMode)
	return threads, nil
}

func wireK8sClient(ctx context.Context) (k8s.Client, error) {
	clientConfig := k8s.ProvideClientConfig()
	config, err := k8s.ProvideKubeConfig(clientConfig)
	if err != nil {
		return nil, err
	}
	env := k8s.ProvideEnv(config)
	portForwarder := k8s.ProvidePortForwarder()
	namespace := k8s.ProvideConfigNamespace(clientConfig)
	kubeContext, err := k8s.ProvideKubeContext(config)
	if err != nil {
		return nil, err
	}
	kubectlRunner := k8s.ProvideKubectlRunner(kubeContext)
	k8sClient := k8s.ProvideK8sClient(ctx, env, portForwarder, namespace, kubectlRunner, clientConfig)
	return k8sClient, nil
}

func wireKubeContext(ctx context.Context) (k8s.KubeContext, error) {
	clientConfig := k8s.ProvideClientConfig()
	config, err := k8s.ProvideKubeConfig(clientConfig)
	if err != nil {
		return "", err
	}
	kubeContext, err := k8s.ProvideKubeContext(config)
	if err != nil {
		return "", err
	}
	return kubeContext, nil
}

func wireKubeConfig(ctx context.Context) (*api.Config, error) {
	clientConfig := k8s.ProvideClientConfig()
	config, err := k8s.ProvideKubeConfig(clientConfig)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func wireEnv(ctx context.Context) (k8s.Env, error) {
	clientConfig := k8s.ProvideClientConfig()
	config, err := k8s.ProvideKubeConfig(clientConfig)
	if err != nil {
		return "", err
	}
	env := k8s.ProvideEnv(config)
	return env, nil
}

func wireNamespace(ctx context.Context) (k8s.Namespace, error) {
	clientConfig := k8s.ProvideClientConfig()
	namespace := k8s.ProvideConfigNamespace(clientConfig)
	return namespace, nil
}

func wireRuntime(ctx context.Context) (container.Runtime, error) {
	clientConfig := k8s.ProvideClientConfig()
	config, err := k8s.ProvideKubeConfig(clientConfig)
	if err != nil {
		return "", err
	}
	env := k8s.ProvideEnv(config)
	portForwarder := k8s.ProvidePortForwarder()
	namespace := k8s.ProvideConfigNamespace(clientConfig)
	kubeContext, err := k8s.ProvideKubeContext(config)
	if err != nil {
		return "", err
	}
	kubectlRunner := k8s.ProvideKubectlRunner(kubeContext)
	k8sClient := k8s.ProvideK8sClient(ctx, env, portForwarder, namespace, kubectlRunner, clientConfig)
	runtime := k8s.ProvideContainerRuntime(ctx, k8sClient)
	return runtime, nil
}

func wireK8sVersion(ctx context.Context) (*version.Info, error) {
	clientConfig := k8s.ProvideClientConfig()
	config, err := k8s.ProvideRESTConfig(clientConfig)
	if err != nil {
		return nil, err
	}
	clientset, err := k8s.ProvideClientSet(config)
	if err != nil {
		return nil, err
	}
	info, err := k8s.ProvideServerVersion(clientset)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func wireDockerVersion(ctx context.Context) (types.Version, error) {
	clientConfig := k8s.ProvideClientConfig()
	config, err := k8s.ProvideKubeConfig(clientConfig)
	if err != nil {
		return types.Version{}, err
	}
	env := k8s.ProvideEnv(config)
	portForwarder := k8s.ProvidePortForwarder()
	namespace := k8s.ProvideConfigNamespace(clientConfig)
	kubeContext, err := k8s.ProvideKubeContext(config)
	if err != nil {
		return types.Version{}, err
	}
	kubectlRunner := k8s.ProvideKubectlRunner(kubeContext)
	k8sClient := k8s.ProvideK8sClient(ctx, env, portForwarder, namespace, kubectlRunner, clientConfig)
	runtime := k8s.ProvideContainerRuntime(ctx, k8sClient)
	minikubeClient := minikube.ProvideMinikubeClient()
	dockerEnv, err := docker.ProvideEnv(ctx, env, runtime, minikubeClient)
	if err != nil {
		return types.Version{}, err
	}
	clientClient, err := docker.ProvideDockerClient(ctx, dockerEnv)
	if err != nil {
		return types.Version{}, err
	}
	typesVersion, err := docker.ProvideDockerVersion(ctx, clientClient)
	if err != nil {
		return types.Version{}, err
	}
	return typesVersion, nil
}

func wireDockerEnv(ctx context.Context) (docker.Env, error) {
	clientConfig := k8s.ProvideClientConfig()
	config, err := k8s.ProvideKubeConfig(clientConfig)
	if err != nil {
		return docker.Env{}, err
	}
	env := k8s.ProvideEnv(config)
	portForwarder := k8s.ProvidePortForwarder()
	namespace := k8s.ProvideConfigNamespace(clientConfig)
	kubeContext, err := k8s.ProvideKubeContext(config)
	if err != nil {
		return docker.Env{}, err
	}
	kubectlRunner := k8s.ProvideKubectlRunner(kubeContext)
	k8sClient := k8s.ProvideK8sClient(ctx, env, portForwarder, namespace, kubectlRunner, clientConfig)
	runtime := k8s.ProvideContainerRuntime(ctx, k8sClient)
	minikubeClient := minikube.ProvideMinikubeClient()
	dockerEnv, err := docker.ProvideEnv(ctx, env, runtime, minikubeClient)
	if err != nil {
		return docker.Env{}, err
	}
	return dockerEnv, nil
}

func wireDownDeps(ctx context.Context) (DownDeps, error) {
	analytics, err := provideAnalytics()
	if err != nil {
		return DownDeps{}, err
	}
	clientConfig := k8s.ProvideClientConfig()
	config, err := k8s.ProvideKubeConfig(clientConfig)
	if err != nil {
		return DownDeps{}, err
	}
	env := k8s.ProvideEnv(config)
	portForwarder := k8s.ProvidePortForwarder()
	namespace := k8s.ProvideConfigNamespace(clientConfig)
	kubeContext, err := k8s.ProvideKubeContext(config)
	if err != nil {
		return DownDeps{}, err
	}
	kubectlRunner := k8s.ProvideKubectlRunner(kubeContext)
	k8sClient := k8s.ProvideK8sClient(ctx, env, portForwarder, namespace, kubectlRunner, clientConfig)
	runtime := k8s.ProvideContainerRuntime(ctx, k8sClient)
	minikubeClient := minikube.ProvideMinikubeClient()
	dockerEnv, err := docker.ProvideEnv(ctx, env, runtime, minikubeClient)
	if err != nil {
		return DownDeps{}, err
	}
	dockerComposeClient := dockercompose.NewDockerComposeClient(dockerEnv)
	tiltfileLoader := tiltfile.ProvideTiltfileLoader(analytics, dockerComposeClient)
	downDeps := ProvideDownDeps(tiltfileLoader, dockerComposeClient, k8sClient)
	return downDeps, nil
}

// wire.go:

var K8sWireSet = wire.NewSet(k8s.ProvideEnv, k8s.DetectNodeIP, k8s.ProvideKubeContext, k8s.ProvideKubeConfig, k8s.ProvideClientConfig, k8s.ProvideClientSet, k8s.ProvideRESTConfig, k8s.ProvidePortForwarder, k8s.ProvideConfigNamespace, k8s.ProvideKubectlRunner, k8s.ProvideContainerRuntime, k8s.ProvideServerVersion, k8s.ProvideK8sClient)

var BaseWireSet = wire.NewSet(
	K8sWireSet, docker.ProvideDockerClient, docker.ProvideDockerVersion, docker.DefaultClient, wire.Bind(new(docker.Client), new(docker.Cli)), dockercompose.NewDockerComposeClient, build.NewImageReaper, tiltfile.ProvideTiltfileLoader, engine.DeployerWireSet, engine.NewPodLogManager, engine.NewPortForwardController, engine.NewBuildController, engine.NewPodWatcher, engine.NewServiceWatcher, engine.NewImageController, engine.NewConfigsController, engine.NewDockerComposeEventWatcher, engine.NewDockerComposeLogManager, engine.NewProfilerManager, engine.NewGithubClientFactory, engine.NewTiltVersionChecker, provideClock, hud.NewRenderer, hud.NewDefaultHeadsUpDisplay, provideLogActions, store.NewStore, wire.Bind(new(store.RStore), new(store.Store)), provideTiltInfo, engine.ProvideSubscribers, engine.NewUpper, provideAnalytics, engine.ProvideAnalyticsReporter, provideUpdateModeFlag, engine.NewWatchManager, engine.ProvideFsWatcherMaker, engine.ProvideTimerMaker, provideWebVersion,
	provideWebMode,
	provideWebURL,
	provideWebPort,
	provideWebDevPort, server.ProvideHeadsUpServer, assets.ProvideAssetServer, server.ProvideHeadsUpServerController, provideSailMode,
	provideSailURL, client.SailWireSet, provideThreads, engine.NewKINDPusher,
)

type Threads struct {
	hud       hud.HeadsUpDisplay
	upper     engine.Upper
	tiltBuild model.TiltBuild
	sailMode  model.SailMode
}

func provideThreads(h hud.HeadsUpDisplay, upper engine.Upper, b model.TiltBuild, sailMode model.SailMode) Threads {
	return Threads{h, upper, b, sailMode}
}

type DownDeps struct {
	tfl      tiltfile.TiltfileLoader
	dcClient dockercompose.DockerComposeClient
	kClient  k8s.Client
}

func ProvideDownDeps(
	tfl tiltfile.TiltfileLoader,
	dcClient dockercompose.DockerComposeClient,
	kClient k8s.Client) DownDeps {
	return DownDeps{
		tfl:      tfl,
		dcClient: dcClient,
		kClient:  kClient,
	}
}

func provideClock() func() time.Time {
	return time.Now
}
