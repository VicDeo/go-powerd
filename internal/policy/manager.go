package policy

// Manager is a struct that manages the policies.
type Manager struct {
	Name     string
	Policies []*Policy
}

// Update updates the policies.
// It is called when the battery level or charging status changes.
func (m *Manager) Update(current int) {
	for _, p := range m.Policies {
		p.Evaluate(current)
	}
}

// ResetAll resets all the policies.
// It is called when the power source changes.
func (m *Manager) ResetAll() {
	for _, p := range m.Policies {
		p.Reset()
	}
}
