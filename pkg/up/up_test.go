package up

import (
	"testing"

	"github.com/jbrekelmans/kube-compose/pkg/config"
	
	version "github.com/hashicorp/go-version"
)

func newTestUpRunner() *upRunner {
	serviceA := &config.Service{
		ServiceName: "a",
	}
	serviceB := &config.Service{
		ServiceName: "b",
	}
	serviceC := &config.Service{
		ServiceName: "c",
	}
	serviceD := &config.Service{
		ServiceName: "d",
	}
	serviceA.DependsOn = map[*config.Service]config.ServiceHealthiness{
		serviceB: config.ServiceHealthy,
	}
	serviceB.DependsOn = map[*config.Service]config.ServiceHealthiness{
		serviceC: config.ServiceHealthy,
		serviceD: config.ServiceHealthy,
	}
	cfg := &config.Config{
		CanonicalComposeFile: config.CanonicalComposeFile{
			Services: map[string]*config.Service{
				serviceA.ServiceName: serviceA,
				serviceB.ServiceName: serviceB,
				serviceC.ServiceName: serviceC,
				serviceD.ServiceName: serviceD,
			},
			Version: version.Must(version.NewVersion("2.3")),
		},
	}

	u := &upRunner{
		cfg: cfg,
	}
	return u
}

func TestComputeDependsOnClosureSuccess(t *testing.T) {
	u := newTestUpRunner()
	u.initApps()
	appA := u.apps["a"]
	appB := u.apps["b"]
	appC := u.apps["c"]
	appD := u.apps["d"]
	
	result := map[*app]bool{}
	
	// Since appA depends on appB, and appB depends on appC and appD, we expect the result to contain all 4 apps.
	u.computeDependsOnClosure(appA, result)
	_, resultContainsAppA := result[appA]
	_, resultContainsAppB := result[appB]
	_, resultContainsAppC := result[appC]
	_, resultContainsAppD := result[appD]
	if !resultContainsAppA || !resultContainsAppB || !resultContainsAppC || !resultContainsAppD {
		t.Fail()
	}
}

func TestComputeDependsOnClosureDoesNotRecompute(t *testing.T) {
	u := newTestUpRunner()
	u.initApps()
	appA := u.apps["a"]
	appB := u.apps["b"]
	
	result := map[*app]bool{}

	// computeDependsOnClosure assumes that if an app is already included in result, then it was from a previous call to
	// computeDependsOnClosure. We test that computeDependsOnClosure does not recompute the dependencies of apps already included in
	// result.
	result[appB] = true

	u.computeDependsOnClosure(appA, result)
	_, resultContainsAppA := result[appA]
	_, resultContainsAppB := result[appB]
	if !resultContainsAppA || !resultContainsAppB {
		t.Fail()
	}
}

func TestInitAppToBeStartedEmpty(t *testing.T) {
	u := newTestUpRunner()
	u.initApps()

	// If len(u.cfg.Services) == 0 then alls app should be started.
	u.initAppsToBeStarted()
	for _, app := range u.apps {
		if _, ok := u.appsToBeStarted[app]; !ok {
			t.Fail()
		}
	}
}

func TestInitAppToBeStartedNonEmpty(t *testing.T) {
	u := newTestUpRunner()
	u.initApps()

	appC := u.apps["c"]
	u.cfg.Services = map[string]bool{
		"c": true,
	}
	u.initAppsToBeStarted()
	
	if len(u.appsToBeStarted) != 1 {
		t.Fail()
	}
	if _, ok := u.appsToBeStarted[appC]; !ok {
		t.Fail()	
	}
}