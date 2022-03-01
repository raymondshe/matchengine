package raftadmin

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/Jille/raft-grpc-leader-rpc/rafterrors"
	"github.com/hashicorp/raft"
	"github.com/raymondshe/matchengine/proto"
	pb "github.com/raymondshe/matchengine/proto"
)

// WordTracker keeps track of the three longest words it ever saw.
type WordTracker struct {
	mtx   sync.RWMutex
	words [3]string
}

var _ raft.FSM = &WordTracker{}

// compareWords returns true if a is longer (lexicography breaking ties).
func compareWords(a, b string) bool {
	if len(a) == len(b) {
		return a < b
	}
	return len(a) > len(b)
}

func cloneWords(words [3]string) []string {
	var ret [3]string
	copy(ret[:], words[:])
	return ret[:]
}

func (f *WordTracker) Apply(l *raft.Log) interface{} {
	f.mtx.Lock()
	defer f.mtx.Unlock()
	w := string(l.Data)
	for i := 0; i < len(f.words); i++ {
		if compareWords(w, f.words[i]) {
			copy(f.words[i+1:], f.words[i:])
			f.words[i] = w
			break
		}
	}
	return nil
}

func (f *WordTracker) Snapshot() (raft.FSMSnapshot, error) {
	// Make sure that any future calls to f.Apply() don't change the snapshot.
	return &snapshot{cloneWords(f.words)}, nil
}

func (f *WordTracker) Restore(r io.ReadCloser) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	words := strings.Split(string(b), "\n")
	copy(f.words[:], words)
	return nil
}

type snapshot struct {
	words []string
}

func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	_, err := sink.Write([]byte(strings.Join(s.words, "\n")))
	if err != nil {
		sink.Cancel()
		return fmt.Errorf("sink.Write(): %v", err)
	}
	return sink.Close()
}

func (s *snapshot) Release() {
}

type RpcInterface struct {
	WordTracker *WordTracker
	Raft        *raft.Raft
	proto.UnimplementedExampleServer
}

func (r RpcInterface) AddWord(ctx context.Context, req *pb.AddWordRequest) (*pb.AddWordResponse, error) {
	f := r.Raft.Apply([]byte(req.GetWord()), time.Second)
	if err := f.Error(); err != nil {
		return nil, rafterrors.MarkRetriable(err)
	}
	return &pb.AddWordResponse{
		CommitIndex: f.Index(),
	}, nil
}

func (r RpcInterface) GetWords(ctx context.Context, req *pb.GetWordsRequest) (*pb.GetWordsResponse, error) {
	r.WordTracker.mtx.RLock()
	defer r.WordTracker.mtx.RUnlock()
	return &pb.GetWordsResponse{
		BestWords:   cloneWords(r.WordTracker.words),
		ReadAtIndex: r.Raft.AppliedIndex(),
	}, nil
}
