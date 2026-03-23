// policy is the package to deal with the policies.
package policy

// Policy is a struct that represents a policy.
type Policy struct {
	Name            string
	Threshold       int  // The target (e.g., 20% for low, 95% for full)
	Hysteresis      int  // The "buffer" to prevent flickering (e.g., 5%)
	TriggerOnRising bool // True = Charging case, False = Discharging case

	IsTriggered bool
	OnTrigger   func()
	OnReset     func()
}

// Evaluate evaluates the policy. It is called when the battery level changes.
func (p *Policy) Evaluate(current int) {
	if p.TriggerOnRising {
		// CHARGING case (e.g., Alert at 95%, Reset at 90%)
		if !p.IsTriggered && current >= p.Threshold {
			p.IsTriggered = true
			if p.OnTrigger != nil {
				p.OnTrigger()
			}
		} else if p.IsTriggered && current <= (p.Threshold-p.Hysteresis) {
			p.Reset()
		}
	} else {
		// DISCHARGING case (e.g., Alert at 20%, Reset at 25%)
		if !p.IsTriggered && current <= p.Threshold {
			p.IsTriggered = true
			if p.OnTrigger != nil {
				p.OnTrigger()
			}
		} else if p.IsTriggered && current >= (p.Threshold+p.Hysteresis) {
			p.Reset()
		}
	}
}

// Reset resets the policy. It is called when the policy is no longer triggered.
func (p *Policy) Reset() {
	if p.IsTriggered {
		p.IsTriggered = false
		if p.OnReset != nil {
			p.OnReset()
		}
	}
}
