package watches

import (
	"context"
	"fmt"
	"log"

	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

type Handler func(event *watch.Event)

type watchedResource struct {
	gvr       *schema.GroupVersionResource
	name      string
	namespace string
}

type Watches struct {
	dc        dynamic.Interface
	handler   Handler
	resources []*watchedResource
	watches   []watch.Interface
	started   bool
}

func New(dc dynamic.Interface, eventHandler Handler) *Watches {
	return &Watches{dc: dc, handler: eventHandler}
}

func (w *Watches) Add(resource, name, namespace string) error {
	gvr, _ := schema.ParseResourceArg(resource)
	if gvr == nil {
		return fmt.Errorf("invalid resource %s", resource)
	}

	r := &watchedResource{
		gvr:       gvr,
		name:      name,
		namespace: namespace,
	}

	w.resources = append(w.resources, r)
	return nil
}

func (w *Watches) Start() error {
	if w.started {
		return fmt.Errorf("invalid state, watches already started")
	}

	for _, r := range w.resources {
		if err := w.startWatch(r); err != nil {
			return err
		}
	}

	w.started = true
	return nil
}

func (w *Watches) startWatch(r *watchedResource) error {
	opts := buildListOptions(r.name, r.namespace)
	watch, err := w.dc.Resource(*r.gvr).Watch(context.Background(), opts)
	if err != nil {
		return errors.Wrapf(err, "failed to watch resource %+v", r)
	}

	w.watches = append(w.watches, watch)

	go func() {
		for e := range watch.ResultChan() {
			w.handler(&e)
		}
	}()

	return nil
}

func buildListOptions(name, namespace string) v1.ListOptions {
	opts := v1.ListOptions{}

	fieldSelector := ""
	if name != "" {
		fieldSelector = fmt.Sprintf("metadata.name=%s", name)
	}

	if namespace != "" {
		if fieldSelector == "" {
			fieldSelector = fmt.Sprintf("metadata.namespace=%s", namespace)
		} else {
			fieldSelector = fmt.Sprintf("%s,metadata.namespace=%s", fieldSelector, namespace)
		}
	}

	if fieldSelector != "" {
		log.Printf("using fieldselector '%s'", fieldSelector)
		opts.FieldSelector = fieldSelector
	}

	return opts
}

func (w *Watches) Stop() {
	for _, watch := range w.watches {
		watch.Stop()
	}

	if w.watches != nil {
		w.watches = w.watches[:0]
	}

	w.started = false
}
