package app

import (
	"sync"
	"time"

	"github.com/thin-edge/tedge-container-plugin/pkg/container"
)

// mergeFilterOptions merges two FilterOptions for the UpdateDebouncer.
//
// When either operand has no specific IDs or Names (a "full scan"), that
// operand's FilterOptions is returned so that its global exclusion/inclusion
// filters are preserved and a full doUpdate() is triggered.
//
// When both are scoped, the IDs and Names are unioned and the global filter
// fields from the first operand are kept (they originate from the same base
// CLI filter options).
func mergeFilterOptions(a, b container.FilterOptions) container.FilterOptions {
	aScoped := len(a.IDs) > 0 || len(a.Names) > 0
	bScoped := len(b.IDs) > 0 || len(b.Names) > 0

	if !aScoped {
		return a
	}
	if !bScoped {
		return b
	}

	// Both are scoped — union IDs and Names; keep global filters from a.
	seenIDs := make(map[string]struct{}, len(a.IDs)+len(b.IDs))
	ids := make([]string, 0, len(a.IDs)+len(b.IDs))
	for _, id := range append(a.IDs, b.IDs...) {
		if _, ok := seenIDs[id]; !ok {
			seenIDs[id] = struct{}{}
			ids = append(ids, id)
		}
	}

	seenNames := make(map[string]struct{}, len(a.Names)+len(b.Names))
	names := make([]string, 0, len(a.Names)+len(b.Names))
	for _, n := range append(a.Names, b.Names...) {
		if _, ok := seenNames[n]; !ok {
			seenNames[n] = struct{}{}
			names = append(names, n)
		}
	}

	return container.FilterOptions{
		IDs:              ids,
		Names:            names,
		Labels:           a.Labels,
		Types:            a.Types,
		ExcludeNames:     a.ExcludeNames,
		ExcludeWithLabel: a.ExcludeWithLabel,
	}
}

// mergeRequests merges two ActionRequests. For ActionUpdateAll the
// FilterOptions are merged with mergeFilterOptions. For all other action
// types, or when the actions differ, the broader/earlier request is kept.
func mergeRequests(a, b ActionRequest) ActionRequest {
	if a.Action == ActionUpdateAll && b.Action == ActionUpdateAll {
		merged := mergeFilterOptions(
			a.Options.(container.FilterOptions),
			b.Options.(container.FilterOptions),
		)
		return ActionRequest{Action: ActionUpdateAll, Options: merged}
	}
	// If a is a full UpdateAll keep it as it is the most permissive.
	if a.Action == ActionUpdateAll {
		opts := a.Options.(container.FilterOptions)
		if len(opts.IDs) == 0 && len(opts.Names) == 0 {
			return a
		}
	}
	return b
}

// UpdateDebouncer coalesces rapid-fire ActionUpdateAll requests into a single
// execution after a configurable quiet period. This prevents a
// crash-looping container (with restart policy "always") from causing a
// flood of doUpdate calls.
//
// Requests received during the debounce window are merged: scoped requests
// have their IDs/Names unioned, and a full-scan request always supersedes
// any scoped request.
type UpdateDebouncer struct {
	mu       sync.Mutex
	pending  *ActionRequest
	timer    *time.Timer
	delay    time.Duration
	dispatch func(ActionRequest)
}

// NewUpdateDebouncer creates a debouncer with the given quiet-period delay.
// dispatch is called with the merged request once the quiet period elapses
// after the last Enqueue call.
func NewUpdateDebouncer(delay time.Duration, dispatch func(ActionRequest)) *UpdateDebouncer {
	return &UpdateDebouncer{
		delay:    delay,
		dispatch: dispatch,
	}
}

// Enqueue adds req to the pending set. If a request is already waiting it is
// merged with req and the debounce timer is reset.
func (d *UpdateDebouncer) Enqueue(req ActionRequest) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.pending == nil {
		d.pending = &req
	} else {
		merged := mergeRequests(*d.pending, req)
		d.pending = &merged
	}

	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.delay, d.fire)
}

func (d *UpdateDebouncer) fire() {
	d.mu.Lock()
	r := d.pending
	d.pending = nil
	d.timer = nil
	d.mu.Unlock()
	if r != nil {
		d.dispatch(*r)
	}
}
