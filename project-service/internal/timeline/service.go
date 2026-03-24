package timeline

type Service struct {
	timelineRepo Repository
}

func NewProjectTimelineService(timelineRepo Repository) *Service {
	return &Service{
		timelineRepo: timelineRepo,
	}
}

type TimelineRepo interface {
}
