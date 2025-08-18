package graphql

import (
	"fmt"
	"stormlink/server/ent"
	"sync"
)

// Храним для каждой подписки map[subscriberID]chan *ent.Comment
var (
  commentAddedSubs   = map[int]map[string]chan *ent.Comment{}
  commentUpdatedSubs = map[int]map[string]chan *ent.Comment{}
  // глобальные каналы для общей ленты
  commentAddedGlobalSubs   = map[string]chan *ent.Comment{}
  commentUpdatedGlobalSubs = map[string]chan *ent.Comment{}
  subsMu             = sync.RWMutex{}
  nextSubID          = 0
)

func subscribeCommentAdded(postID int) (string, <-chan *ent.Comment) {
  subsMu.Lock()
  defer subsMu.Unlock()
  if commentAddedSubs[postID] == nil {
    commentAddedSubs[postID] = make(map[string]chan *ent.Comment)
  }
  id := fmt.Sprintf("add-%d", nextSubID)
  nextSubID++
  ch := make(chan *ent.Comment, 1)
  commentAddedSubs[postID][id] = ch
  return id, ch
}

func unsubscribeCommentAdded(postID int, subID string) {
  subsMu.Lock()
  defer subsMu.Unlock()
  delete(commentAddedSubs[postID], subID)
}

func publishCommentAdded(postID int, comment *ent.Comment) {
  subsMu.RLock()
  defer subsMu.RUnlock()
  for _, ch := range commentAddedSubs[postID] {
    select {
    case ch <- comment:
    default:
    }
  }
  // также оповещаем глобальных подписчиков
  for _, ch := range commentAddedGlobalSubs {
    select {
    case ch <- comment:
    default:
    }
  }
}

// Глобальная подписка
func subscribeCommentAddedGlobal() (string, <-chan *ent.Comment) {
  subsMu.Lock()
  defer subsMu.Unlock()
  id := fmt.Sprintf("global-%d", nextSubID)
  nextSubID++
  ch := make(chan *ent.Comment, 1)
  commentAddedGlobalSubs[id] = ch
  return id, ch
}

func unsubscribeCommentAddedGlobal(subID string) {
	subsMu.Lock()
	defer subsMu.Unlock()
	delete(commentAddedGlobalSubs, subID)
}

// Глобальная подписка на обновления комментариев
func subscribeCommentUpdatedGlobal() (string, <-chan *ent.Comment) {
	subsMu.Lock()
	defer subsMu.Unlock()
	id := fmt.Sprintf("update-global-%d", nextSubID)
	nextSubID++
	ch := make(chan *ent.Comment, 1)
	commentUpdatedGlobalSubs[id] = ch
	return id, ch
}

func unsubscribeCommentUpdatedGlobal(subID string) {
	subsMu.Lock()
	defer subsMu.Unlock()
	delete(commentUpdatedGlobalSubs, subID)
}

// Функции для подписки на обновления комментариев
func subscribeCommentUpdated(postID int) (string, <-chan *ent.Comment) {
	subsMu.Lock()
	defer subsMu.Unlock()
	if commentUpdatedSubs[postID] == nil {
		commentUpdatedSubs[postID] = make(map[string]chan *ent.Comment)
	}
	id := fmt.Sprintf("update-%d", nextSubID)
	nextSubID++
	ch := make(chan *ent.Comment, 1)
	commentUpdatedSubs[postID][id] = ch
	return id, ch
}

func unsubscribeCommentUpdated(postID int, subID string) {
	subsMu.Lock()
	defer subsMu.Unlock()
	delete(commentUpdatedSubs[postID], subID)
}

func publishCommentUpdated(postID int, comment *ent.Comment) {
	subsMu.RLock()
	defer subsMu.RUnlock()
	for _, ch := range commentUpdatedSubs[postID] {
		select {
		case ch <- comment:
		default:
		}
	}
	// всегда оповещаем глобальных подписчиков (в т.ч. при hasDeleted = true)
	for _, ch := range commentUpdatedGlobalSubs {
		select {
		case ch <- comment:
		default:
		}
	}
}
