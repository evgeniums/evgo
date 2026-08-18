package filedata_service

import (
	"context"
	"testing"

	"github.com/evgeniums/evgo/pkg/api"
	"github.com/evgeniums/evgo/pkg/filestorage"
)

// fakeController is the minimal FileDataController NewFileDataService() needs at
// construction time - it only ever calls controller.UrlManager() while building the
// resource tree (Registry()/UploadPart()/Fetch() are never touched by the
// constructor), so those are left unimplemented.
type fakeController struct {
	urlManager filestorage.UrlManager
}

func (c *fakeController) Registry() filestorage.FileInfoRegistry { return nil }
func (c *fakeController) UrlManager() filestorage.UrlManager     { return c.urlManager }
func (c *fakeController) UploadPart(ctx context.Context) error   { return nil }
func (c *fakeController) Fetch(ctx context.Context) error        { return nil }

func newTestUrlManagerForRoutes(t *testing.T, enableTopic bool) *filestorage.UrlManagerBase {
	t.Helper()
	u := filestorage.NewUrlManager()
	u.ID_PARAMETER = "id"
	u.PART_PARAMETER = "part"
	u.TOPIC_PARAMETER = "topic"
	u.ENABLE_TOPIC = enableTopic
	return u
}

// Task debug-sending-files-to-optimized-music (server stage 2, A+B): regression
// test for two stacked defects in the old NewIdResourcesChain - AddOperation()
// used to attach the fetch/upload operation to the HEAD of the resource chain,
// leaving the :id/:part children (and hence the operation) unreachable in the
// route tree at all, and even fixed, the chain rendered as /:topic/:id/:part
// (HasId REPLACES a literal segment, not appends to it) instead of
// /topic/:topic/id/:id/part/:part. Both meant the server registered
// "POST /filedata/upload" - a bare 3-segment path with no id/part params -
// instead of the 6-segment path the client's signed URLs actually use.
func TestFileDataServiceRoutePaths(t *testing.T) {

	cases := []struct {
		name           string
		enableTopic    bool
		wantFetchPath  string
		wantUploadPath string
	}{
		{
			name:           "topic_enabled",
			enableTopic:    true,
			wantFetchPath:  "/filedata/fetch/topic/:topic/id/:id",
			wantUploadPath: "/filedata/upload/topic/:topic/id/:id/part/:part",
		},
		{
			name:           "topic_disabled",
			enableTopic:    false,
			wantFetchPath:  "/filedata/fetch/id/:id",
			wantUploadPath: "/filedata/upload/id/:id/part/:part",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			controller := &fakeController{urlManager: newTestUrlManagerForRoutes(t, c.enableTopic)}
			s := NewFileDataService(controller)

			if s.fetch.Resource().FullPathPrototype() != c.wantFetchPath {
				t.Fatalf("fetch path = %q, want %q", s.fetch.Resource().FullPathPrototype(), c.wantFetchPath)
			}
			if s.upload.Resource().FullPathPrototype() != c.wantUploadPath {
				t.Fatalf("upload path = %q, want %q", s.upload.Resource().FullPathPrototype(), c.wantUploadPath)
			}

			// The operation must actually be attached to the path-carrying leaf,
			// not just some ancestor - the original defect (A) left the head
			// with the operation but the leaf (the resource whose own
			// FullPathPrototype is the full 6/4-segment path above) with none,
			// so ResourceBase.EachOperation() never emitted a route past the
			// head's own literal segment.
			seen := 0
			err := s.fetch.Resource().EachOperation(func(op api.Operation) error {
				if op.Name() == FetchName {
					seen++
				}
				return nil
			})
			if err != nil {
				t.Fatalf("EachOperation failed: %v", err)
			}
			if seen != 1 {
				t.Fatalf("fetch leaf resource has %d operations named %q, want 1", seen, FetchName)
			}
		})
	}
}
