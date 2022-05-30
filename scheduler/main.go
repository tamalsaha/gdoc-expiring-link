package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/madflojo/tasks"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	"github.com/syndtr/goleveldb/leveldb"
)

type Scheduler struct {
	db *leveldb.DB
	s  *tasks.Scheduler
}

func NewScheduler() (*Scheduler, error) {
	path, _ := ioutil.TempDir("", "tasks")
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &Scheduler{
		s:  tasks.New(),
		db: db,
	}, nil
}

func (s *Scheduler) Close() error {
	s.s.Stop()
	return s.db.Close()
}

func (s *Scheduler) Cleanup(fn func(interface{}) error) error {
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		// Remember that the contents of the returned slice should not be modified, and
		// only valid until the next call to Next.
		key := iter.Key()
		args := iter.Value()

		if t, _ := s.s.Lookup(string(key)); t != nil {
			continue
		}

		if err := fn(args); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to execute cleanup function for task id %s, args %s, err: %v", string(key), string(args), err)
		}
		if err := s.db.Delete(key, nil); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to delete key %s, args %s, err: %v", string(key), string(args), err)
		}
	}
	iter.Release()
	return iter.Error()
}

func (s *Scheduler) Schedule(t time.Time, fn func([]byte) error, args []byte) error {
	for {
		id := xid.New()
		err := s.s.AddWithID(id.String(), &tasks.Task{
			Interval: time.Until(t),
			RunOnce:  true,
			TaskFunc: func() error {
				if err := fn(args); err != nil {
					return err
				}
				return s.db.Delete(id.Bytes(), nil)
			},
			ErrFunc: func(e error) {
				_ = s.db.Delete(id.Bytes(), nil)
				_, _ = fmt.Fprintf(os.Stderr, "an error occurred when executing task %s - %v", id, e)
			},
		})
		if err == tasks.ErrIDInUse {
			continue
		} else if err != nil {
			return errors.Wrapf(err, "failed to schedule task")
		}
		if err = s.db.Put(id.Bytes(), args, nil); err != nil {
			return err
		}
		break
	}
	return nil
}

// https://github.com/madflojo/tasks/blob/main/tasks_test.go
func main() {
	// Start the Scheduler
	scheduler := tasks.New()
	defer scheduler.Stop()

	// Add a one time only task for 60 seconds from now
	id, err := scheduler.Add(&tasks.Task{
		Interval: 15 * time.Second,
		RunOnce:  true,
		TaskFunc: func() error {
			fmt.Println(">>>>")
			return nil
		},
		ErrFunc: func(e error) {
			log.Printf("An error occurred when executing task %v", e)
		},
	})
	if err != nil {
		// Do Stuff
	}
	fmt.Println(id)

	// select {}
	time.Sleep(1 * time.Hour)

	//// Add a task
	//id, err := scheduler.Add(&tasks.Task{
	//	Interval: time.Duration(30 * time.Second),
	//	TaskFunc: func() error {
	//		// Put your logic here
	//	}(),
	//})
	//if err != nil {
	//	// Do Stuff
	//}
}
