package types

import "time"

type Attribute struct {
	Name  string      `json:"name"`
	Value interface{} `json:"value"`
}

type LinkType string

func NewLinkType(linkType string) LinkType {
	return LinkType(linkType)
}

func (l LinkType) String() string {
	return string(l)
}

type EventLink struct {
	EventID    string       `json:"event_id"`
	EventType  string       `json:"event_type"`
	LinkType   LinkType     `json:"link_type"`
	Attributes []*Attribute `json:"attributes"`
}

type Event struct {
	ID         string       `json:"id"`
	Type       string       `json:"type"`
	Timestamp  time.Time    `json:"timestamp"`
	Attributes []*Attribute `json:"attributes"`
	Links      []*EventLink `json:"links"`
}
