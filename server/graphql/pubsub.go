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
  // глобальный канал для общей ленты
  commentAddedGlobalSubs = map[string]chan *ent.Comment{}
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

// Аналогично для commentUpdated:
// func subscribeCommentUpdated(postID int) (string, <-chan *ent.Comment) { /*…*/ }
// func unsubscribeCommentUpdated(postID int, subID string)       { /*…*/ }
// func publishCommentUpdated(postID int, comment *ent.Comment)  { /*…*/ }
