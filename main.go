package main

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"

	"github.com/nirmata/kube-events/pkg/watches"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	cfg := ctrl.GetConfigOrDie()
	dc, err := dynamic.NewForConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}

	c := cache.New(5*time.Minute, 5*time.Minute)

	h := func(e *watch.Event) {
		u := e.Object.(*unstructured.Unstructured)
		log.Printf("received event: %s %s %s", e.Type, u.GetKind(), u.GetName())

		k := makeKey(u)
		v, ok := c.Get(k)
		if v != nil && ok {
			prior := v.(*unstructured.Unstructured)

			p, err := createPatch(u, prior)
			if err != nil {
				log.Printf("failed to create patch: %v", err)
			} else {
				log.Printf("got patch %s %s", k, string(p))
			}
		}

		c.Set(k, u, cache.DefaultExpiration)
	}

	watches := watches.New(dc, h)

	if err := watches.Add("pods.v1.", "", ""); err != nil {
		log.Fatal(err)
	}

	if err := watches.Add("events.v1.", "", ""); err != nil {
		log.Fatal(err)
	}

	if err := watches.Add("deployments.v1.apps", "", ""); err != nil {
		log.Fatal(err)
	}

	if err := watches.Add("namespaces.v1.", "", ""); err != nil {
		log.Fatal(err)
	}

	if err := watches.Start(); err != nil {
		log.Fatal(err)
	}

	<-make(chan int)
}

func makeKey(u *unstructured.Unstructured) string {
	toks := make([]string, 4)

	toks[0] = u.GetAPIVersion()
	toks[1] = u.GetKind()
	toks[2] = u.GetNamespace()
	toks[3] = u.GetName()

	return strings.Join(toks, "/")
}

func createPatch(o *unstructured.Unstructured, n *unstructured.Unstructured) ([]byte, error) {
	oBytes, err := json.Marshal(o.Object)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshall prior obj %s", makeKey(o))
	}

	nBytes, err := json.Marshal(n.Object)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshall new obj", makeKey(n))
	}

	patch, err := jsonpatch.CreateMergePatch(oBytes, nBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create patch")
	}

	return patch, nil
}
