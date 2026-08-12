package domain

// CanTransition reports whether a status change is allowed.
func CanTransition(from, to Status) bool {
	if from == to {
		return false
	}
	allowed, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return allowed == to
}

// NextStatus returns the next status in the workflow, if any.
func NextStatus(current Status) (Status, bool) {
	next, ok := allowedTransitions[current]
	return next, ok
}

var allowedTransitions = map[Status]Status{
	StatusOpen:          StatusAcknowledged,
	StatusAcknowledged:  StatusInvestigating,
	StatusInvestigating: StatusMitigated,
	StatusMitigated:     StatusResolved,
	StatusResolved:      StatusClosed,
}
