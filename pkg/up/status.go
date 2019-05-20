package up

type podStatus int

const (
	podStatusReady           podStatus = 2
	podStatusStarted         podStatus = 1
	podStatusOther           podStatus = 0
	podStatusCompleted       podStatus = 3
	podStatusReadyString               = "ready"
	podStatusStartedString             = "started"
	podStatusCompletedString           = "completed"
	podStatusOtherString               = "other"
)

func (podStatus *podStatus) String() string {
	switch *podStatus {
	case podStatusReady:
		return podStatusReadyString
	case podStatusStarted:
		return podStatusStartedString
	case podStatusCompleted:
		return podStatusCompletedString
	}
	return podStatusOtherString
}
