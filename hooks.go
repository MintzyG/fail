package fail

// OnErrorCreated registers a hook that runs whenever an error is created
func OnErrorCreated(hook func(*Error)) {
	global.OnErrorCreated(hook)
}

func (r *Registry) OnErrorCreated(hook func(*Error)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onErrorCreated = append(r.onErrorCreated, hook)
}

func (r *Registry) runOnCreate(err *Error) {
	for _, hook := range r.onErrorCreated {
		hook(err)
	}
}
