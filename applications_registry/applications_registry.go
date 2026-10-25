package applicationsregistry

import (
	"context"
	"errors"
	"fmt"
	"onibi/application"
	"sync"
)

// interchangeble with sql
type ApplicationStore interface {
	Add(ctx context.Context, app *application.Application) error
	Get(ctx context.Context, ID string) (application.Application, error)
	List(ctx context.Context, OwnerID string) ([]application.Application, error)
	Update(ctx context.Context, app *application.Application) error
	Transition(ctx context.Context, // returning app because when release mutex something could happen with it
		id, owner string, to application.Status) (application.Application, error)
}

type InMemoryApplicationStore struct {
	mu           sync.RWMutex
	applications map[string]*application.Application
}

func NewInMemoryApplicationStore() *InMemoryApplicationStore {
	return &InMemoryApplicationStore{applications: make(map[string]*application.Application)}
}

func (r *InMemoryApplicationStore) List(ctx context.Context, ownerID string) ([]application.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	resarr := make([]application.Application, 0)
	for _, v := range r.applications {
		if v.OwnerID == ownerID {
			resarr = append(resarr, *v)
		}
	}
	return resarr, nil
}

func (r *InMemoryApplicationStore) Get(ctx context.Context,
	ID string) (application.Application, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.applications[ID]
	if !ok {
		return application.Application{}, fmt.Errorf("no application with this ID")
	}
	return *v, nil
}

func (r *InMemoryApplicationStore) Add(ctx context.Context,
	app *application.Application) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	app, err := application.NewApplicaton(app.Inn,
		app.Loan, app.Term, app.Goal, app.OwnerID)
	if err != nil {
		return err
	}
	r.applications[app.ID] = app
	return nil
}

// don't really know if we need this
func (r *InMemoryApplicationStore) Delete(ctx context.Context, ID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.applications[ID]; !ok {
		return fmt.Errorf("no such application")
	}
	delete(r.applications, ID)
	return nil
}

func (r *InMemoryApplicationStore) Update(
	ctx context.Context, app *application.Application) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	// change enter-fields if status draft
	return nil
}

var (
	NotFound error = errors.New("not found")
)

func (r *InMemoryApplicationStore) Transition(ctx context.Context,
	id, owner string,
	to application.Status) (application.Application, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	app, ok := r.applications[id]
	if !ok || owner != app.OwnerID {
		return application.Application{}, NotFound
	}
	if err := app.Transition(to); err != nil {
		return application.Application{}, err
	}
	return *app, nil
}
