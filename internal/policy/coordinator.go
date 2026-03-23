package policy

type Coordinator struct {
	ChargingMngr    *Manager
	DischargingMngr *Manager
	ActiveMngr      *Manager
	LastStatus      string // "Charging", "Discharging", etc.
}

func (c *Coordinator) HandleUpdate(capacity int, status string) {
	if status != c.LastStatus {
		if c.ActiveMngr != nil {
			c.ActiveMngr.ResetAll()
		}

		if status == "Charging" || status == "Full" {
			c.ActiveMngr = c.ChargingMngr
		} else {
			c.ActiveMngr = c.DischargingMngr
		}
		c.LastStatus = status
	}

	if c.ActiveMngr != nil {
		c.ActiveMngr.Update(capacity)
	}
}
