package service

import "github.com/joshp123/xuezh/internal/xuezh/events"

type EventRecord struct {
	EventID   string
	EventType string
	TS        string
	Modality  string
	Items     []string
	Context   *string
}

type LogEventOptions struct {
	EventType string
	Modality  string
	Items     []string
	Context   *string
}

func (App) LogEvent(opts LogEventOptions) (EventRecord, error) {
	event, err := events.LogEvent(opts.EventType, opts.Modality, opts.Items, opts.Context)
	if err != nil {
		return EventRecord{}, err
	}
	return eventRecord(event), nil
}

func (App) ListEvents(since string, limit int) ([]EventRecord, error) {
	items, err := events.ListEvents(since, limit)
	if err != nil {
		return nil, err
	}
	result := make([]EventRecord, 0, len(items))
	for _, event := range items {
		result = append(result, eventRecord(event))
	}
	return result, nil
}

func eventRecord(event events.Event) EventRecord {
	return EventRecord{
		EventID:   event.EventID,
		EventType: event.EventType,
		TS:        event.TS,
		Modality:  event.Modality,
		Items:     event.Items,
		Context:   event.Context,
	}
}
