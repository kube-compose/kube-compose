package up

type podStatus int

const (
	podStatusReady     podStatus = 2
	podStatusStarted   podStatus = 1
	podStatusOther     podStatus = 0
	podStatusCompleted podStatus = 3
)

func (podStatus *podStatus) String() string {
	switch *podStatus {
	case podStatusReady:
		return "ready"
	case podStatusStarted:
		return "started"
	case podStatusCompleted:
		return "completed"
	}
	return "other"
}
