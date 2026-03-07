package pubsub_subscriber

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/evgeniums/evgo/pkg/app_context"
	"github.com/evgeniums/evgo/pkg/generic_error"
	"github.com/evgeniums/evgo/pkg/message"
	"github.com/evgeniums/evgo/pkg/message/message_json"
	"github.com/evgeniums/evgo/pkg/op_context"
	"github.com/evgeniums/evgo/pkg/op_context/default_op_context"
	"github.com/evgeniums/evgo/pkg/utils"
)

type Subscriber interface {
	Subscribe(topic Topic) (string, error)
	Unsubscribe(topicName string, subscriptionId ...string)
	Shutdown(ctx context.Context) error

	Handle(ctx op_context.Context, topicName string, msg []byte) error
}

type WithSubscriber interface {
	Subscriber() Subscriber
}

type SubscriberBase struct {
	app_context.WithAppBase
	mutex      sync.RWMutex
	topics     map[string]map[string]Topic
	serializer message.Serializer
}

func (s *SubscriberBase) Construct(app app_context.Context, serializer ...message.Serializer) {
	s.WithAppBase.Init(app)
	s.topics = make(map[string]map[string]Topic)
	s.serializer = utils.OptionalArg(message.Serializer(message_json.Serializer), serializer...)
}

func (s *SubscriberBase) Topics(topicName string) (map[string]Topic, error) {
	s.mutex.RLock()
	topics, ok := s.topics[topicName]
	s.mutex.RUnlock()
	if !ok {
		return nil, errors.New("topics not found")
	}
	return topics, nil
}

func (s *SubscriberBase) Handle(ctx op_context.Context, topicName string, msg []byte) error {
	s.mutex.RLock()
	topics, ok := s.topics[topicName]
	s.mutex.RUnlock()
	if !ok {
		return nil
	}
	for _, topic := range topics {
		err := topic.Handle(ctx, msg, s.serializer)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SubscriberBase) AddTopic(topic Topic) (string, error) {

	subscriptionId := utils.GenerateID()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	topics, exists := s.topics[topic.Name()]
	if !exists {
		topics = make(map[string]Topic)
	}
	topics[subscriptionId] = topic
	s.topics[topic.Name()] = topics

	return subscriptionId, nil
}

func (s *SubscriberBase) DeleteTopic(topicName string, subscriptionId ...string) bool {

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(subscriptionId) == 0 {
		delete(s.topics, topicName)
		return true
	}

	topics, ok := s.topics[topicName]
	if !ok {
		return true
	}

	delete(topics, subscriptionId[0])
	if len(topics) == 0 {
		delete(s.topics, topicName)
		return true
	}

	s.topics[topicName] = topics
	return false
}

func (s *SubscriberBase) NewOpContext(topicName string) op_context.Context {

	opCtx := default_op_context.NewContext()
	opCtx.Init(s.App(), s.App().Logger(), s.App().Db())
	opCtx.SetName(topicName)
	errManager := &generic_error.ErrorManagerBase{}
	errManager.Init(http.StatusInternalServerError)
	opCtx.SetErrorManager(errManager)

	origin := default_op_context.NewOrigin(s.App())
	origin.SetUserType("pubsub")
	opCtx.SetOrigin(origin)

	return opCtx
}
