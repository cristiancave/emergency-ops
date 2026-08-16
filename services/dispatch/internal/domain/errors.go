package domain

import "errors"

var (
	ErrAmbulanceNotFound       = errors.New("ambulance not found")
	ErrDispatchNotFound        = errors.New("dispatch not found")
	ErrNoAvailableAmbulance    = errors.New("no available ambulance matching priority")
	ErrDuplicateDispatchID     = errors.New("dispatch with this ID already exists")
)