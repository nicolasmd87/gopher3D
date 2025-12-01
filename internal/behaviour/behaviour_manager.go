package behaviour

type PlayerBehaviour interface {
	Start()
	Update()
	UpdateFixed()
}

type BehaviourWrapper struct {
	Behaviour PlayerBehaviour
	started   bool
}

type BehaviourManager struct {
	behaviours []BehaviourWrapper
}

var GlobalBehaviourManager = NewBehaviourManager()

func NewBehaviourManager() *BehaviourManager {
	return &BehaviourManager{}
}

func (m *BehaviourManager) Add(behaviour PlayerBehaviour) {
	m.behaviours = append(m.behaviours, BehaviourWrapper{Behaviour: behaviour, started: false})
}

func (m *BehaviourManager) Remove(behaviour PlayerBehaviour) {
	// Find and remove the behaviour
	for i := range m.behaviours {
		if m.behaviours[i].Behaviour == behaviour {
			// Remove by swapping with last element and truncating
			m.behaviours[i] = m.behaviours[len(m.behaviours)-1]
			m.behaviours = m.behaviours[:len(m.behaviours)-1]
			return
		}
	}
}

// Clear removes all behaviours from the manager
func (m *BehaviourManager) Clear() {
	m.behaviours = m.behaviours[:0]
}

func (m *BehaviourManager) UpdateAll() {
	// Update old behaviour system
	for i := range m.behaviours {
		if !m.behaviours[i].started {
			m.behaviours[i].Behaviour.Start()
			m.behaviours[i].started = true
		}
		m.behaviours[i].Behaviour.Update()
	}

	// Update new component system
	GlobalComponentManager.UpdateAll()
}

func (m *BehaviourManager) UpdateAllFixed() {
	// Update old behaviour system
	for i := range m.behaviours {
		if !m.behaviours[i].started {
			m.behaviours[i].Behaviour.Start()
			m.behaviours[i].started = true
		}
		m.behaviours[i].Behaviour.UpdateFixed()
	}

	// Update new component system
	GlobalComponentManager.FixedUpdateAll()
}
