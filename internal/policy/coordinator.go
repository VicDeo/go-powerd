package policy

type Coordinator struct {
	ChargingMngr    *Manager
	DischargingMngr *Manager
	ActiveMngr      *Manager
	LastStatus      bool // true when there is power, false when not
}

func (c *Coordinator) HandleUpdate(capacity int, isPluggedIn bool) {
	if isPluggedIn != c.LastStatus {
		if c.ActiveMngr != nil {
			c.ActiveMngr.ResetAll()
		}

		if isPluggedIn {
			c.ActiveMngr = c.ChargingMngr
		} else {
			c.ActiveMngr = c.DischargingMngr
		}
		c.LastStatus = isPluggedIn
	}

	if c.ActiveMngr != nil {
		c.ActiveMngr.Update(capacity)
	}
}
