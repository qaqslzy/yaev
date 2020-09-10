package poller

import "sync"

/**
*
* @author Liu Weiyi
* @date 2020/7/30 1:33 下午
 */

type noteQueue struct {
	mu    sync.Mutex
	notes []interface{}
}

func (q *noteQueue) Add(note interface{}) (one bool) {
	q.mu.Lock()

	q.notes = append(q.notes, note)
	n := len(q.notes)
	q.mu.Unlock()
	return n == 1
}

func (q *noteQueue) ForEach(iter func(note interface{}) error) error {
	q.mu.Lock()
	if len(q.notes) == 0 {
		q.mu.Unlock()
		return nil
	}
	notes := q.notes
	q.notes = nil
	q.mu.Unlock()
	for _, note := range notes{
		if err:= iter(note); err != nil{
			return err
		}
	}
	//q.mu.Lock()
	//if q.notes == nil {
	//	for i := range notes {
	//		notes[i] = nil
	//	}
	//	q.notes = notes[:0]
	//}
	//q.mu.Unlock()
	return nil
}