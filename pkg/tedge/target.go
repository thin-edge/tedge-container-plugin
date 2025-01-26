package tedge

import (
	"fmt"
	"strings"
)

type Target struct {
	RootPrefix    string
	TopicID       string
	CloudIdentity string
}

func (t *Target) ExternalID() string {
	if t.TopicID == "device/main//" {
		return t.CloudIdentity
	}
	return strings.TrimRight(t.CloudIdentity+":"+strings.ReplaceAll(t.TopicID, "/", ":"), ":")
}

func (t *Target) Topic() string {
	return GetTopic(*t)
}

func (t *Target) Service(name string) *Target {
	target := NewTarget(t.RootPrefix, strings.Join(strings.Split(t.TopicID, "/")[0:2], "/")+"/service/"+name)
	target.CloudIdentity = t.CloudIdentity
	return target
}

func NewTarget(rootPrefix, topicID string) *Target {
	if rootPrefix == "" {
		rootPrefix = "te"
	}
	return &Target{
		RootPrefix: rootPrefix,
		TopicID:    topicID,
	}
}

func NewTargetFromTopic(topic string) (*Target, error) {
	parts := strings.Split(topic, "/")
	if len(parts) >= 5 {
		return &Target{
			RootPrefix: parts[0],
			TopicID:    strings.Join(parts[1:5], "/"),
		}, nil
	}
	return nil, fmt.Errorf("invalid topic")
}

func GetTopicRegistration(target Target) string {
	return GetTopic(target)
}

func GetHealthTopic(target Target) string {
	return GetTopic(target, "status", "health")
}

func GetTopic(target Target, subpath ...string) string {
	if len(subpath) == 0 {
		return fmt.Sprintf("%s/%s", target.RootPrefix, target.TopicID)
	}
	return fmt.Sprintf("%s/%s/%s", target.RootPrefix, target.TopicID, strings.Join(subpath, "/"))
}
